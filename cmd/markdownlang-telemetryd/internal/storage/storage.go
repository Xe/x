// Package storage implements the S3/Tigris storage layer for telemetry reports.
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"within.website/x/store"
	"within.website/x/tigris"
)

const (
	// envBucketName is the environment variable for the bucket name
	envBucketName = "MARKDOWNLANG_TELEMETRY_BUCKET"
)

var (
	// bucketName is the S3 bucket name for storing telemetry reports
	bucketName = os.Getenv(envBucketName)
)

// Report contains telemetry data to store.
// This mirrors the Report structure from the markdownlang telemetry package.
type Report struct {
	// System information
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	GoVersion string `json:"go_version"`
	NumCPU    int    `json:"num_cpu"`
	Hostname  string `json:"hostname,omitempty"`
	UnameAll  string `json:"uname_all,omitempty"`

	// Git configuration
	GitUserName  string `json:"git_user_name,omitempty"`
	GitUserEmail string `json:"git_user_email,omitempty"`

	// Program information
	Version       string `json:"version"`
	Program       string `json:"program"`
	ProgramSHA256 string `json:"program_sha256,omitempty"`

	// Execution metrics
	DurationMs    int64    `json:"duration_ms"`
	ToolCallCount int      `json:"tool_call_count"`
	ToolsUsed     []string `json:"tools_used"`
	MCPServers    []string `json:"mcp_servers,omitempty"`
	MCPToolsUsed  []string `json:"mcp_tools_used,omitempty"`

	// Model information
	ModelProviderURL string `json:"model_provider_url,omitempty"`
	ModelName        string `json:"model_name,omitempty"`

	// Environment details
	Shell      string `json:"shell,omitempty"`
	Term       string `json:"term,omitempty"`
	Timezone   string `json:"timezone,omitempty"`
	WorkingDir string `json:"working_dir,omitempty"`

	// Timestamp
	Timestamp int64 `json:"timestamp"`
}

// Storage handles storing telemetry reports in S3/Tigris.
type Storage struct {
	store  store.Interface
	s3     *s3.Client
	bucket string
}

// New creates a new Storage client.
// If MARKDOWNLANG_TELEMETRY_BUCKET is not set, a type 7 UUID will be generated as the bucket name.
// The bucket will be created if it doesn't exist.
func New(ctx context.Context) (*Storage, error) {
	// Generate bucket name if not set
	if bucketName == "" {
		bucketName = uuid.Must(uuid.NewV7()).String()
	}

	// Create S3 client for bucket operations
	s3Client, err := tigris.Client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create tigris client: %w", err)
	}

	// Create bucket if it doesn't exist
	_, err = s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// Ignore error if bucket already exists
		var bucketAlreadyExists *types.BucketAlreadyExists
		var bucketAlreadyOwnedByYou *types.BucketAlreadyOwnedByYou
		if !(errors.As(err, &bucketAlreadyExists) || errors.As(err, &bucketAlreadyOwnedByYou)) {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	// Create store Interface for object operations
	st, err := store.NewS3API(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to create s3api store: %w", err)
	}

	return &Storage{
		store:  st,
		s3:     s3Client,
		bucket: bucketName,
	}, nil
}

// Store saves a telemetry report to S3.
// The report is stored as JSON in a bucket, organized by email address: "{email}/{timestamp}.json".
func (s *Storage) Store(ctx context.Context, report Report) error {
	// Marshal report to JSON
	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	// Generate object key: {email}/{timestamp}.json
	email := report.GitUserEmail
	if email == "" {
		email = "unknown"
	}
	timestamp := time.Unix(report.Timestamp, 0).UTC().Format("2006-01-02T15-04-05")
	key := fmt.Sprintf("%s/%s.json", email, timestamp)

	// Store the report using the store Interface
	if err := s.store.Set(ctx, key, data); err != nil {
		return fmt.Errorf("failed to store report: %w", err)
	}

	return nil
}

// BucketName returns the bucket name being used.
func (s *Storage) BucketName() string {
	return s.bucket
}

// Interface returns the underlying store.Interface for use by other components.
func (s *Storage) Interface() store.Interface {
	return s.store
}
