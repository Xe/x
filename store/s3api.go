package store

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func NewS3API(ctx context.Context, bucket string) (Interface, error) {
	cfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("can't load AWS config from environment: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = false // Tigris needs vhost style
	})

	return &S3API{
		s3:     client,
		bucket: bucket,
	}, nil
}

type S3API struct {
	s3     *s3.Client
	bucket string
}

func (s *S3API) Delete(ctx context.Context, key string) error {
	// Emulate not found by probing first.
	if _, err := s.s3.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &s.bucket, Key: &key}); err != nil {
		return fmt.Errorf("%w: %w", ErrNotFound, err)
	}
	iopsMetrics.WithLabelValues("s3api", "HeadObject")
	if _, err := s.s3.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucket, Key: &key}); err != nil {
		return fmt.Errorf("can't delete from s3: %w", err)
	}
	iopsMetrics.WithLabelValues("s3api", "DeleteObject")
	return nil
}

func (s *S3API) Exists(ctx context.Context, key string) error {
	_, err := s.s3.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &s.bucket, Key: &key})
	iopsMetrics.WithLabelValues("s3api", "HeadObject")
	if err != nil {
		return fmt.Errorf("%w: %w", ErrNotFound, err)
	}
	return nil
}

func (s *S3API) Get(ctx context.Context, key string) ([]byte, error) {
	out, err := s.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	iopsMetrics.WithLabelValues("s3api", "GetObject")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotFound, err)
	}
	defer out.Body.Close()

	b, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("can't read s3 object: %w", err)
	}
	return b, nil
}

func (s *S3API) Set(ctx context.Context, key string, value []byte) error {
	_, err := s.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
		Body:   bytes.NewReader(value),
	})
	iopsMetrics.WithLabelValues("s3api", "PutObject")
	if err != nil {
		return fmt.Errorf("can't put s3 object: %w", err)
	}
	return nil
}

func (s *S3API) List(ctx context.Context, prefix string) ([]string, error) {
	items, err := s.s3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &s.bucket,
		Prefix: aws.String(prefix),
	})
	iopsMetrics.WithLabelValues("s3api", "ListObjectsV2")
	if err != nil {
		return nil, fmt.Errorf("can't list items: %w", err)
	}

	var result []string

	for _, item := range items.Contents {
		result = append(result, *item.Key)
	}

	return result, nil
}
