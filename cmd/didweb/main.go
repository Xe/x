package main

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/multiformats/go-multibase"
	"github.com/whyrusleeping/go-did"
	"within.website/x/internal"
	"within.website/x/web"
)

var (
	curveName   = flag.String("curve-name", "p256", "elliptic curve to use")
	handleName  = flag.String("handle-name", "", "The bluesky handle you want to use")
	pdsHostname = flag.String("pds-hostname", "", "The Personal Data Server hostname for this account")
)

func main() {
	internal.HandleStartup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	crv, ok := getCurve(*curveName)
	if !ok {
		log.Fatalf("invalid curve: %s", *curveName)
	}

	if *handleName == "" {
		log.Fatal("pass --handle-name")
	}

	if *pdsHostname == "" {
		log.Fatal("pass --pds-hostname (no scheme)")
	}

	privkey, err := generateKey(crv)
	if err != nil {
		log.Fatalf("can't make privkey: %v", err)
	}

	privkeyString := serializePrivateKey(privkey)

	pubkeyString, err := serializePublicKey(crv, privkey)
	if err != nil {
		log.Fatalf("can't serialize public key: %v", err)
	}

	fmt.Printf("------\nFor did:web:%s:\npublic:  %s\nprivate: %s\n", *handleName, pubkeyString, privkeyString)

	handleDID, err := did.ParseDID("did:web:" + *handleName)
	if err != nil {
		log.Fatalf("can't parse your DID: %v", err)
	}

	pdsDID, err := did.ParseDID("did:web:" + *pdsHostname)
	if err != nil {
		log.Fatalf("can't parse PDS did: %v", err)
	}

	serviceDID, err := did.ParseDID("#atproto_pds")
	if err != nil {
		log.Fatalf("[unexpected] can't parse service DID: %v", err)
	}

	didDoc := Document{
		Context: []string{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/multikey/v1",
			"https://w3id.org/security/suites/secp256k1-2019/v1",
		},
		Id:          handleDID,
		AlsoKnownAs: []string{"at://" + *handleName},
		VerificationMethod: []*VerificationMethod{
			{
				ID:                 handleDID.String() + "#atproto",
				Type:               "Multikey",
				Controller:         handleDID.String(),
				PublicKeyMultibase: pubkeyString,
			},
		},
		Service: []*Service{
			{
				ID:              serviceDID,
				Type:            "AtprotoPersonalDataServer",
				ServiceEndpoint: "https://" + *pdsHostname,
			},
		},
	}

	fout, err := os.Create("did.json")
	if err != nil {
		log.Fatalf("can't make did.json: %v", err)
	}

	enc := json.NewEncoder(fout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(didDoc); err != nil {
		fout.Close()
		log.Fatalf("can't encode DID doc: %v", err)
	}

	if err := fout.Close(); err != nil {
		log.Fatalf("can't close did.json: %v", err)
	}

	fmt.Printf("upload did.json to https://%[1]s/.well-known/did.json and press enter ...\n(hint: make sure %[1]s points to where you uploaded it to!)\n", *handleName)
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	if err := waitUntilDIDWorks(ctx, *handleName, pubkeyString); err != nil {
		log.Fatalf("%s doesn't work: %v", handleDID.String(), err)
	}

	fmt.Println("DID validated!")

	if err := os.WriteFile("atproto-did", []byte(handleDID.String()), 0600); err != nil {
		log.Fatalf("can't write atproto-did: %v", err)
	}

	fmt.Printf("upload atproto-did to https://%s/.well-known/atproto-did in it, then press enter\n", handleDID.Value())
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	cli, err := mkAccount(ctx, privkey, &handleDID, &pdsDID)
	if err != nil {
		log.Fatalf("can't make account: %v", err)
	}

	credSuggestions, err := IdentityGetRecommendedDidCredentials(ctx, cli)
	if err != nil {
		fmt.Printf("access token: %s\n", cli.Auth.AccessJwt)
		fmt.Println("You will need to do things manually, sorry")
		log.Fatalf("can't get suggested credentials: %v", err)
	}

	// give the PDS authority over the identity
	didDoc.VerificationMethod[0].PublicKeyMultibase = credSuggestions.VerificationMethods.Atproto.Value()

	fout, err = os.Create("did.json")
	if err != nil {
		log.Fatalf("can't make did.json: %v", err)
	}

	enc = json.NewEncoder(fout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(didDoc); err != nil {
		fout.Close()
		log.Fatalf("can't encode DID doc: %v", err)
	}

	if err := fout.Close(); err != nil {
		log.Fatalf("can't close did.json: %v", err)
	}

	fmt.Printf("re-upload did.json to https://%[1]s/.well-known/did.json and press enter ...\n", *handleName)
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	if err := waitUntilDIDWorks(ctx, *handleName, credSuggestions.VerificationMethods.Atproto.Value()); err != nil {
		log.Fatalf("%s doesn't work: %v", handleDID.String(), err)
	}

	if err := atproto.ServerActivateAccount(ctx, cli); err != nil {
		log.Fatalf("can't activate account: %v", err)
	}

	fmt.Println("have fun skeeting!")
}

func getCurve(name string) (elliptic.Curve, bool) {
	switch name {
	case "p256":
		return elliptic.P256(), true
	default:
		return nil, false
	}
}

func generateKey(crv elliptic.Curve) (*ecdsa.PrivateKey, error) {
	privkey, err := ecdsa.GenerateKey(crv, rand.Reader)
	if err != nil {
		return nil, err
	}

	return privkey, nil
}

func serializePrivateKey(privkey *ecdsa.PrivateKey) string {
	return hex.EncodeToString(privkey.D.Bytes())
}

func serializePublicKey(crv elliptic.Curve, privkey *ecdsa.PrivateKey) (string, error) {
	// varint encoded version of 0x1200, see https://atproto.com/specs/cryptography#public-key-encoding
	var varintP256 = []byte{0x80, 0x24}

	b := slices.Concat(varintP256, elliptic.MarshalCompressed(crv, privkey.PublicKey.X, privkey.PublicKey.Y))
	pubkey, err := multibase.Encode(multibase.Base58BTC, b)
	if err != nil {
		return "", err
	}

	return pubkey, nil
}

func fetchDIDWeb(ctx context.Context, domain string) (*did.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://%s/.well-known/did.json", domain), nil)
	if err != nil {
		return nil, fmt.Errorf("can't construct request for domain %s: %w", domain, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can't connect to domain %s: %w", domain, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result did.Document
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("can't decode DID document on %s: %w", domain, err)
	}

	return &result, nil
}

func waitUntilDIDWorks(ctx context.Context, domain, wantPubkey string) error {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	errCount := 0

	for range t.C {
		if errCount >= 30 {
			return fmt.Errorf("gave up after %d tries", errCount)
		}

		doc, err := fetchDIDWeb(ctx, domain)
		if err != nil {
			errCount++
			slog.Error("can't load did doc", "domain", domain, "err", err)
			continue
		}

		var multiKey *did.VerificationMethod
		for _, vm := range doc.VerificationMethod {
			if vm.Type == "Multikey" {
				multiKey = &vm
			}
		}

		if multiKey == nil {
			return fmt.Errorf("invalid DID, need Multikey verification method")
		}

		if theirKey := *multiKey.PublicKeyMultibase; theirKey != wantPubkey {
			return fmt.Errorf("wrong public key: want %s, got %s", wantPubkey, theirKey)
		}

		return nil
	}

	return fmt.Errorf("how did you get here?")
}

func makeJWTFor(privkey *ecdsa.PrivateKey, lxm, aud, iss string, exp time.Duration) (string, error) {
	type myClaims struct {
		jwt.RegisteredClaims
		Lexicon string `json:"lxm"` // Atproto XRPC lexicon method for the scope of this JWT
	}

	jwt.MarshalSingleStringAsArray = false
	claims := myClaims{
		Lexicon: lxm,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    iss,
			Audience:  jwt.ClaimStrings{aud},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tokenString, err := token.SignedString(privkey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func mkAccount(ctx context.Context, privkey *ecdsa.PrivateKey, handle, pds *did.DID) (*xrpc.Client, error) {
	didJWT, err := makeJWTFor(privkey, "com.atproto.server.createAccount", pds.String(), handle.String(), 180*time.Second)
	if err != nil {
		return nil, err
	}

	cli := &xrpc.Client{
		Auth: &xrpc.AuthInfo{
			AccessJwt: didJWT,
			Handle:    handle.Value(),
			Did:       handle.String(),
		},
		Host: "https://" + pds.Value(),
	}

	fmt.Printf("hint: generate an invite code with:\n$ sudo pdsadmin create-invite-code\n\n")

	inviteCode := input("PDS invite code")
	email := input("email")
	password := input("password")

	inp := &atproto.ServerCreateAccount_Input{
		Did:        &[]string{handle.String()}[0],
		Email:      &email,
		Handle:     handle.Value(),
		InviteCode: &inviteCode,
		Password:   &password,
	}

	resp, err := atproto.ServerCreateAccount(ctx, cli, inp)
	if err != nil {
		return nil, err
	}

	cli.Auth.AccessJwt = resp.AccessJwt
	cli.Auth.RefreshJwt = resp.RefreshJwt
	cli.Auth.Did = resp.Did
	cli.Auth.Handle = resp.Handle

	return cli, nil
}

func input(prompt string) string {
	fmt.Printf("%s> ", prompt)
	text, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(text)
}
