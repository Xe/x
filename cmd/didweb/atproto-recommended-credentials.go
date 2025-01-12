package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bluesky-social/indigo/xrpc"
	"github.com/whyrusleeping/go-did"
	"within.website/x/web"
)

type RecommendedCredentials struct {
	AlsoKnownAs         []string `json:"alsoKnownAs"`
	VerificationMethods struct {
		Atproto did.DID `json:"atproto"`
	} `json:"verificationMethods"`
	RotationKeys []string `json:"rotationKeys"`
	Services     struct {
		AtprotoPds struct {
			Type     string `json:"type"`
			Endpoint string `json:"endpoint"`
		} `json:"atproto_pds"`
	} `json:"services"`
}

func IdentityGetRecommendedDidCredentials(ctx context.Context, cli *xrpc.Client) (*RecommendedCredentials, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cli.Host+"/xrpc/com.atproto.identity.getRecommendedDidCredentials", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+cli.Auth.AccessJwt)
	req.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result RecommendedCredentials
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
