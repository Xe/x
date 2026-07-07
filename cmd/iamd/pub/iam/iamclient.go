package iam

import (
	"context"
	"net/http"

	iamv1 "within.website/x/gen/within/website/x/iam/v1"
	"within.website/x/web/middleware/sigv4a/sigv4aclient"
	"within.website/x/web/useragent"
)

type Client struct {
	Keys  iamv1.KeyService
	Users iamv1.UserService
}

func New(ctx context.Context, endpoint, region, accessKeyID, secretAccessKey string) (*Client, error) {
	ua := useragent.Transport("within.website/x/cmd/iamd/pub/iam", "https://xeiaso.net/contact", http.DefaultTransport)

	rt, err := sigv4aclient.NewSigV4ARoundTripper(&sigv4aclient.Config{
		Region:      region,
		AccessKey:   accessKeyID,
		SecretKey:   secretAccessKey,
		ServiceName: "iam",
	}, ua)
	if err != nil {
		return nil, err
	}

	hc := &http.Client{
		Transport: rt,
	}

	keys := iamv1.NewKeyServiceProtobufClient(endpoint, hc)
	users := iamv1.NewUserServiceProtobufClient(endpoint, hc)

	return &Client{
		Keys:  keys,
		Users: users,
	}, nil
}
