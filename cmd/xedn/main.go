// Command quiche is a little cute cache server for my B2 bucket.
package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strings"

	"github.com/golang/groupcache"
	"github.com/sebest/xff"
	"tailscale.com/tsnet"
	"within.website/ln"
	"within.website/ln/ex"
	"within.website/ln/opname"
	"within.website/x/internal"
	"within.website/x/web"
)

var (
	b2Backend = flag.String("b2-backend", "https://f001.backblazeb2.com", "Backblaze B2 base URL")
	addr      = flag.String("addr", ":8080", "server address")
)

const cacheSize = 128 * 1024 * 1024 // 128 mebibytes

type CacheData struct {
	Headers http.Header
	Body    []byte
}

var Group = groupcache.NewGroup("b2-bucket", cacheSize, groupcache.GetterFunc(
	func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
		ln.Log(context.Background(), ln.F{"key": key})

		resp, err := http.Get(*b2Backend + key)
		if err != nil {
			return fmt.Errorf("can't fetch from b2: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return web.NewError(http.StatusOK, resp)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("can't read from b2: %v", err)
		}

		result := &CacheData{
			Headers: resp.Header,
			Body:    body,
		}

		var buf bytes.Buffer
		err = gob.NewEncoder(&buf).Encode(result)
		if err != nil {
			return err
		}

		dest.SetBytes(buf.Bytes())

		return nil
	},
))

func findLocalPeer(peers []string) (string, error) {
	var addrs []net.Addr
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		ifaceAddrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}

		addrs = append(addrs, ifaceAddrs...)
	}

	for _, addr := range addrs {
		prefix, err := netip.ParsePrefix(addr.String())
		if err != nil {
			return "", err
		}

		for _, peer := range peers {
			ln.Log(context.Background(), ln.F{"peer": peer, "addr": prefix.Addr().String()})
			if strings.Contains(peer, prefix.Addr().String()) {
				return peer, nil
			}
		}
	}

	return "", errors.New("can't find local peer somehow")
}

func discoverPeers() ([]string, error) {
	ips, err := net.LookupIP("xedn.internal")
	if err != nil {
return nil, err
	}

	var result []string

	for _, ip := range ips {
		nip, _ := netip.AddrFromSlice(ip)
		ipp := netip.AddrPortFrom(nip, 8081)
		u, err := url.Parse("http://" + ipp.String())
		if err != nil {
			return nil, err
		}
		result = append(result, u.String())
	}

	return result, nil
}

func main() {
	internal.HandleStartup()
	ctx := opname.With(context.Background(), "startup")

	go func () {
		srv := &tsnet.Server{
			Hostname: "xedn-" + os.Getenv("FLY_REGION"),
			Logf: log.New(io.Discard, "", 0).Printf,
			AuthKey:   os.Getenv("TS_AUTHKEY"),
		}

		lis, err := srv.Listen("tcp", ":80")
		if err != nil {
			ln.FatalErr(ctx, err, ln.Action("tsnet listening"))
		}

		defer srv.Close()
		defer lis.Close()
		ln.FatalErr(opname.With(ctx, "metrics-tsnet"), http.Serve(lis, ex.HTTPLog(http.DefaultServeMux)))
	} ()

	xffMW, err := xff.Default()
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/file/christine-static/", func(w http.ResponseWriter, r *http.Request) {
		var b []byte
		err := Group.Get(nil, r.URL.Path, groupcache.AllocatingByteSliceSink(&b))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		var result CacheData
		err = gob.NewDecoder(bytes.NewBuffer(b)).Decode(&result)
		if err != nil {
			ln.Error(r.Context(), err)
			http.Error(w, "internal cache error", http.StatusInternalServerError)
			return
		}

		for k, vs := range result.Headers {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write(result.Body)
	})

	ln.Log(context.Background(), ln.F{"addr": *addr})
	http.ListenAndServe(*addr, xffMW.Handler(ex.HTTPLog(mux)))
}
