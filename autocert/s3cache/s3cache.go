package s3cache

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	storage "github.com/tigrisdata/storage-go"
	"golang.org/x/crypto/acme/autocert"
)

type Options struct {
	Bucket string
	Prefix string
}

func New(ctx context.Context, opts Options) (autocert.Cache, error) {
	s3c, err := storage.New(ctx, storage.WithGlobalEndpoint(), storage.WithPathStyle(true))
	if err != nil {
		return nil, err
	}

	return &impl{
		cli:    s3c,
		bucket: opts.Bucket,
		prefix: opts.Prefix,
	}, nil
}

type impl struct {
	cli            *storage.Client
	bucket, prefix string
}

func (i *impl) Get(ctx context.Context, key string) ([]byte, error) {
	resp, err := i.cli.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(i.bucket),
		Key:    aws.String(path.Join(i.prefix, key)),
	})
	if err != nil {
		return nil, fmt.Errorf("can't get object at %s: %w", key, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("can't read object at %s: %w", key, err)
	}

	return data, nil
}

func (i *impl) Put(ctx context.Context, key string, data []byte) error {
	if _, err := i.cli.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(i.bucket),
		Key:    aws.String(path.Join(i.prefix, key)),
		Body:   bytes.NewBuffer(data),
	}); err != nil {
		return fmt.Errorf("can't put object at %s: %w", key, err)
	}

	return nil
}

func (i *impl) Delete(ctx context.Context, key string) error {
	if _, err := i.cli.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(i.bucket),
		Key:    aws.String(path.Join(i.prefix, key)),
	}); err != nil {
		return fmt.Errorf("can't delete object at %s: %w", key, err)
	}

	return nil
}
