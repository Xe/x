package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ErrNotFound is returned by Store.Get when the key does not exist.
var ErrNotFound = errors.New("store: key not found")

// Store is the object storage the handlers need.
type Store interface {
	Put(ctx context.Context, key, contentType string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Presign(ctx context.Context, key string) (string, error)
}

type s3Store struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
	expiry  time.Duration
}

func newS3Store(client *s3.Client, bucket string, expiry time.Duration) *s3Store {
	return &s3Store{
		client:  client,
		presign: s3.NewPresignClient(client),
		bucket:  bucket,
		expiry:  expiry,
	}
}

func (s *s3Store) Put(ctx context.Context, key, contentType string, data []byte) error {
	if _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(int64(len(data))),
	}); err != nil {
		return fmt.Errorf("store: can't put %s: %w", key, err)
	}

	return nil
}

func (s *s3Store) Get(ctx context.Context, key string) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("store: can't get %s: %w", key, err)
	}
	defer obj.Body.Close()

	data, err := io.ReadAll(obj.Body)
	if err != nil {
		return nil, fmt.Errorf("store: can't read %s: %w", key, err)
	}

	return data, nil
}

func (s *s3Store) Presign(ctx context.Context, key string) (string, error) {
	req, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(s.expiry))
	if err != nil {
		return "", fmt.Errorf("store: can't presign %s: %w", key, err)
	}

	return req.URL, nil
}
