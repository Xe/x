package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"within.website/x/cmd/iamd/oidc"
)

func loadOIDCConfig(enabled bool, issuer, clientID, secretFile string) (oidc.Config, error) {
	if !enabled {
		return oidc.Config{}, nil
	}
	u, err := url.Parse(issuer)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return oidc.Config{}, fmt.Errorf("OIDC issuer must be an HTTPS origin")
	}
	if clientID == "" || secretFile == "" {
		return oidc.Config{}, fmt.Errorf("enabled OIDC requires an upstream client ID and secret file")
	}
	data, err := os.ReadFile(secretFile)
	if err != nil {
		return oidc.Config{}, fmt.Errorf("read upstream OIDC secret file: %w", err)
	}
	secret := strings.TrimSpace(string(data))
	if secret == "" {
		return oidc.Config{}, fmt.Errorf("upstream OIDC secret file is empty")
	}
	return oidc.Config{Issuer: strings.TrimSuffix(issuer, "/"), UpstreamIssuer: "https://idp.xeserv.us", UpstreamClientID: clientID, UpstreamClientSecret: secret}, nil
}
