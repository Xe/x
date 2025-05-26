package grpchc

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/google/subcommands"
	"github.com/rodaine/table"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type GRPCHealth struct {
	insecure bool
	service  string
}

func (*GRPCHealth) Name() string     { return "grpc-health" }
func (*GRPCHealth) Synopsis() string { return "Run a GRPC health check on the given server" }
func (*GRPCHealth) Usage() string {
	return `grpc-health [--insecure] [--service] <service-url grpc://host:port>
Run a GRPC health probe against a service and exit with non-zero if things are
not healthy.
`
}

func (gh *GRPCHealth) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&gh.insecure, "insecure", false, "If true, connect in insecure mode")
	f.StringVar(&gh.service, "service", "", "Service to check, if empty check the entire server")
}

func (gh *GRPCHealth) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 1 {
		fmt.Println(gh.Usage())
		return subcommands.ExitFailure
	}

	u := f.Arg(0)

	if !strings.HasPrefix(u, "passthrough:///") {
		u = "passthrough:///" + u
	}

	var conn *grpc.ClientConn
	var err error

	switch gh.insecure {
	case true:
		conn, err = grpc.NewClient(u, grpc.WithTransportCredentials(insecure.NewCredentials()))
	case false:
		conn, err = grpc.NewClient(u, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	}
	if err != nil {
		fmt.Printf("can't connect to %s: %v\n", u, err)
		return subcommands.ExitFailure
	}

	health := healthpb.NewHealthClient(conn)

	switch gh.service {
	case "all":
		result, err := health.List(ctx, &healthpb.HealthListRequest{})
		if err != nil {
			fmt.Printf("%v\n", err)
			return subcommands.ExitFailure
		}

		exitCode := subcommands.ExitSuccess

		keys := []string{}
		for svc := range result.GetStatuses() {
			keys = append(keys, svc)
		}
		sort.Strings(keys)

		fmt.Println(u, "status:")

		tbl := table.New("Service", "Status")

		for _, key := range keys {
			status := result.GetStatuses()[key]
			tbl.AddRow(key, status.GetStatus().String())
			if status.GetStatus() != healthpb.HealthCheckResponse_SERVING {
				exitCode = subcommands.ExitFailure
			}
		}

		tbl.Print()

		return exitCode
	default:
		result, err := health.Check(ctx, &healthpb.HealthCheckRequest{
			Service: gh.service,
		})
		if err != nil {
			fmt.Printf("%v\n", err)
			return subcommands.ExitFailure
		}

		fmt.Printf("%s (service %s): %s\n", u, gh.service, result.GetStatus())

		if result.GetStatus() != healthpb.HealthCheckResponse_SERVING {
			return subcommands.ExitFailure
		}
	}

	return subcommands.ExitSuccess
}
