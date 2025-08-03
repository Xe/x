package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"within.website/x/cmd/sakurajima/internal/config"
	"within.website/x/internal/flagenv"
)

var (
	configFname = flag.String("config", "./sakurajima.hcl", "configuration file name")

	ErrNoHandlerDefinedForRoute = errors.New("no handler defined for route")
)

func main() {
	flagenv.Parse()
	flag.Parse()

	var config config.Toplevel
	err := hclsimple.DecodeFile(*configFname, nil, &config)
	if err != nil {
		log.Fatalf("Failed to load configuration: %s", err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(config); err != nil {
		log.Fatal(err)
	}

	var errs []error

	router := mux.NewRouter().StrictSlash(false)

	for _, domain := range config.Domains {
		for _, route := range domain.Routes {
			var h http.Handler

			if route.Folder == "" && route.ReverseProxy == nil {
				errs = append(errs, fmt.Errorf("%w %s", ErrNoHandlerDefinedForRoute, route.Path))
				continue
			}

			if route.Folder != "" {
				if _, err := os.Stat(route.Folder); os.IsNotExist(err) {
					errs = append(errs, fmt.Errorf("folder %q does not exist: %w", route.Folder, err))
					continue
				}

				slog.Info("serving folder", "path", route.Folder, "route", route.Path)
				h = http.StripPrefix(route.Path, http.FileServerFS(os.DirFS(route.Folder)))
			}

			if route.ReverseProxy != nil {
				slog.Info("serving reverse proxy", "target", route.ReverseProxy.Target, "route", route.Path)
				u, err := url.Parse(route.ReverseProxy.Target)
				if err != nil {
					errs = append(errs, fmt.Errorf("can't parse reverse proxy target %q: %w", route.ReverseProxy.Target, err))
					continue
				}

				switch u.Scheme {
				case "http", "https", "https+insecure":
					rp := httputil.NewSingleHostReverseProxy(u)

					if u.Scheme == "https+insecure" {
						slog.Warn("insecure HTTPS target configured", "domain", domain.Name, "route", route.Path)
						rp.Transport = &http.Transport{
							TLSClientConfig: &tls.Config{
								InsecureSkipVerify: true,
							},
						}
					}

					h = rp
				case "h2c":
					h = newH2CReverseProxy(u)
				case "unix":
					h = &httputil.ReverseProxy{
						Transport: &http.Transport{
							DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
								return net.Dial("unix", strings.TrimPrefix(route.ReverseProxy.Target, "unix://"))
							},
						},
					}
				}
			}

			router.Host(domain.Name).Path(route.Path).Handler(h)
			if route.Path != "/" {
				router.Host(domain.Name).Path(route.Path + "/").Handler(h)
				router.Host(domain.Name).PathPrefix(route.Path + "/").Handler(h)
			}
		}
	}

	if len(errs) > 0 {
		for _, err := range errs {
			log.Println(err)
		}
		os.Exit(1)
	}

	log.Fatal(http.ListenAndServe(config.Bind.HTTP, router))
}
