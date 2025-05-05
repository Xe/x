package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"within.website/x/bundler"
	"within.website/x/proto/relayd"
)

var (
	telemetryEnable          = flag.Bool("telemetry-enable", false, "if true, enable request telemetry bundling to object storage")
	telemetryBucket          = flag.String("telemetry-bucket", "relayd-logs", "object storage bucket to dump logs to")
	telemetryPathStyle       = flag.Bool("telemetry-path-style", false, "if true, use s3 path style")
	telemetryHost            = flag.String("telemetry-host", "", "hostname to disambiguate telemetry")
	telemetryBundleCount     = flag.Int("telemetry-bundle-count", 50, "maximum number of items per telemetry bundle")
	telemetryContextDeadline = flag.Duration("telemetry-context-deadline", time.Minute, "maximum time for the telemetry context deadline")
)

type TelemetrySink struct {
	sink *bundler.Bundler[*relayd.RequestLog]
	s3c  *s3.Client
	host string
}

func NewTelemetrySink(ctx context.Context) (*TelemetrySink, error) {
	if !*telemetryEnable {
		return nil, nil
	}

	slog.Info("telemetry enabled")

	cfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	s3c := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	result := &TelemetrySink{
		s3c:  s3c,
		host: *telemetryHost,
	}

	result.sink = bundler.New(result.WriteBundle)
	result.sink.DelayThreshold = time.Minute
	result.sink.BundleCountThreshold = *telemetryBundleCount
	result.sink.ContextDeadline = *telemetryContextDeadline

	return result, nil
}

func (ts *TelemetrySink) Add(item *relayd.RequestLog) {
	ts.sink.Add(item, 1)
}

func (ts *TelemetrySink) WriteBundle(ctx context.Context, items []*relayd.RequestLog) {
	if err := ts.writeBundle(ctx, items); err != nil {
		slog.Error("failed writing", "itemCount", len(items), "err", err)
		for _, item := range items {
			item := item
			// 1 in 8 chance to drop
			if rand.IntN(8) != 4 /* chosen by fair dice roll */ {
				go func(rl *relayd.RequestLog) {
					ts.sink.Add(rl, 1)
				}(item)
			}
		}
	}
}

func (ts *TelemetrySink) writeBundle(ctx context.Context, items []*relayd.RequestLog) error {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)

	for _, item := range items {
		if err := enc.Encode(item); err != nil {
			return err
		}
	}

	id := uuid.Must(uuid.NewV7()).String()

	if _, err := ts.s3c.PutObject(ctx, &s3.PutObjectInput{
		Body:        buf,
		Bucket:      telemetryBucket,
		Key:         aws.String(ts.host + "/" + id + ".jsonl"),
		ContentType: aws.String("application/jsonl"),
	}); err != nil {
		return err
	}

	return nil
}
