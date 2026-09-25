// Package xev contains subcommands that call the xev decision API.
package xev

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"sort"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	xevv1 "within.website/x/gen/xeiaso/net/xev/v1"
)

var (
	xevInsecure = flag.Bool("xev-insecure", true, "Connect to xev without TLS?")
	xevURL      = flag.String("xev-url", "passthrough:///xev.default.svc.alrest.xeserv.us:80", "Base xev URL (gRPC over plaintext HTTP/2)")
)

// Client holds a connection to xev and the service clients that use it.
type Client struct {
	conn     *grpc.ClientConn
	Decision xevv1.DecisionServiceClient
}

// New dials xev and returns a Client. The caller must call Close.
func New() (*Client, error) {
	target := *xevURL
	if !strings.HasPrefix(target, "passthrough:///") {
		target = "passthrough:///" + target
	}

	var dialOpts []grpc.DialOption

	switch *xevInsecure {
	case true:
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	case false:
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	}

	conn, err := grpc.NewClient(target, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("can't connect to xev: %w", err)
	}

	return &Client{
		conn:     conn,
		Decision: xevv1.NewDecisionServiceClient(conn),
	}, nil
}

// Close releases the connection to xev.
func (c *Client) Close() error { return c.conn.Close() }

// parseModel converts a user supplied model name into a Model enum value.
// It accepts "qwen3-600m", "qwen3_600m" and "MODEL_QWEN3_600M". An empty
// string selects MODEL_UNSPECIFIED, which lets xev pick the default model.
func parseModel(name string) (xevv1.Model, error) {
	if name == "" {
		return xevv1.Model_MODEL_UNSPECIFIED, nil
	}

	key := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	if !strings.HasPrefix(key, "MODEL_") {
		key = "MODEL_" + key
	}

	val, ok := xevv1.Model_value[key]
	if !ok {
		return xevv1.Model_MODEL_UNSPECIFIED, fmt.Errorf("unknown model %q, want one of: %s", name, strings.Join(modelNames(), ", "))
	}

	return xevv1.Model(val), nil
}

// modelNames lists the model names that parseModel accepts, in enum order.
func modelNames() []string {
	vals := make([]int32, 0, len(xevv1.Model_name))
	for val := range xevv1.Model_name {
		vals = append(vals, val)
	}
	sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })

	names := make([]string, 0, len(vals))
	for _, val := range vals {
		names = append(names, strings.ToLower(strings.TrimPrefix(xevv1.Model_name[val], "MODEL_")))
	}

	return names
}

// decision is the part of a xev response that both Pick and Noul share.
type decision interface {
	proto.Message
	GetExecutionTime() *durationpb.Duration
	GetConfidence() map[string]float32
}

// output holds the flags that control how a decision is printed.
type output struct {
	all  bool
	json bool
}

// setFlags registers the output flags on f.
func (o *output) setFlags(f *flag.FlagSet) {
	f.BoolVar(&o.all, "all", false, "Print every option with its confidence score.")
	f.BoolVar(&o.json, "json", false, "Print the raw response as JSON.")
}

// score pairs an option name with its confidence score.
type score struct {
	name  string
	value float32
}

// print writes resp to stdout in the format that the output flags select.
func (o *output) print(resp decision) error {
	if o.json {
		buf, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp)
		if err != nil {
			return fmt.Errorf("can't encode response as JSON: %w", err)
		}
		fmt.Println(string(buf))
		return nil
	}

	ranked := rank(resp.GetConfidence())
	if len(ranked) == 0 {
		return errors.New("xev returned no confidence scores")
	}

	if !o.all {
		fmt.Println(ranked[0].name)
		return nil
	}

	for _, sc := range ranked {
		fmt.Printf("%.4f\t%s\n", sc.value, sc.name)
	}
	fmt.Printf("took %s\n", resp.GetExecutionTime().AsDuration())

	return nil
}

// rank sorts confidence scores from highest to lowest. Equal scores sort by
// name so that repeated calls print the same order.
func rank(confidence map[string]float32) []score {
	scores := make([]score, 0, len(confidence))
	for name, value := range confidence {
		scores = append(scores, score{name: name, value: value})
	}

	sort.Slice(scores, func(i, j int) bool {
		if scores[i].value != scores[j].value {
			return scores[i].value > scores[j].value
		}
		return scores[i].name < scores[j].name
	})

	return scores
}
