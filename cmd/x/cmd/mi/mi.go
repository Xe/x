package mi

import (
	"crypto/tls"
	"flag"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	miv1 "within.website/x/gen/within/website/x/mi/v1"
)

var (
	miInsecure = flag.Bool("mi-insecure", true, "Connect to mi without TLS?")
	miURL      = flag.String("mi-url", "passthrough:///mi.mi.svc.alrest.xeserv.us:8081", "Base mi URL (gRPC)")
)

type Client struct {
	conn          *grpc.ClientConn
	Events        miv1.EventsClient
	SwitchTracker miv1.SwitchTrackerClient
}

func New() (*Client, error) {
	var miURL = *miURL
	if !strings.HasPrefix(miURL, "passthrough:///") {
		miURL = "passthrough:///" + miURL
	}

	var dialOpts []grpc.DialOption

	switch *miInsecure {
	case true:
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	case false:
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	}

	conn, err := grpc.NewClient(miURL, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("can't connect to mi: %w", err)
	}

	return &Client{
		conn:          conn,
		Events:        miv1.NewEventsClient(conn),
		SwitchTracker: miv1.NewSwitchTrackerClient(conn),
	}, nil
}
