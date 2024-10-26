package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/repo"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/ipfs/go-cid"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	flag.Parse()
	ctx := context.Background()
	atid, err := syntax.ParseAtIdentifier(flag.Arg(0))
	if err != nil {
		return err
	}

	dir := identity.DefaultDirectory()
	ident, err := dir.Lookup(ctx, *atid)
	if err != nil {
		return err
	}

	if ident.PDSEndpoint() == "" {
		return fmt.Errorf("no PDS endpoint for identity")
	}

	fmt.Println(ident.PDSEndpoint())

	carPath := ident.DID.String() + ".car"

	xrpcc := xrpc.Client{
		Host: ident.PDSEndpoint(),
	}

	repoBytes, err := atproto.SyncGetRepo(ctx, &xrpcc, ident.DID.String(), "")
	if err != nil {
		return err
	}

	err = os.WriteFile(carPath, repoBytes, 0666)
	if err != nil {
		return err
	}

	if err := carList(carPath); err != nil {
		return err
	}

	return nil
}

func carList(carPath string) error {
	ctx := context.Background()
	fi, err := os.Open(carPath)
	if err != nil {
		return fmt.Errorf("failed to open car file: %w", err)
	}
	defer fi.Close()

	// read repository tree into memory
	r, err := repo.ReadRepoFromCar(ctx, fi)
	if err != nil {
		return fmt.Errorf("failed to read repository from car file: %w", err)
	}

	// extract DID from repo commit
	sc := r.SignedCommit()
	did, err := syntax.ParseDID(sc.Did)
	if err != nil {
		return fmt.Errorf("failed to parse DID from signed commit: %w", err)
	}
	topDir := did.String()

	// iterate over all of the records by key and CID
	err = r.ForEach(ctx, "", func(k string, v cid.Cid) error {
		fmt.Printf("%s\t%s\n", k, v.String())

		recPath := topDir + "/" + k
		if err := os.MkdirAll(filepath.Dir(recPath), os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directories for record path: %w", err)
		}

		// fetch the record CBOR and convert to a golang struct
		_, rec, err := r.GetRecord(ctx, k)
		if err != nil {
			return fmt.Errorf("failed to get record for key %s: %w", k, err)
		}

		// serialize as JSON
		recJson, err := json.MarshalIndent(rec, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal record to JSON for key %s: %w", k, err)
		}

		if err := os.WriteFile(recPath+".json", recJson, 0666); err != nil {
			return fmt.Errorf("failed to write JSON file for key %s: %w", k, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to iterate over records: %w", err)
	}
	return nil
}
