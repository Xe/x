// Copyright 2021 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sigv4client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	signer "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var sigv4HeaderDenylist = []string{
	"uber-trace-id",
}

type sigV4RoundTripper struct {
	region      string
	next        http.RoundTripper
	pool        sync.Pool
	creds       *aws.CredentialsCache
	serviceName string
	signer      *signer.Signer
}

// NewSigV4RoundTripper returns a new http.RoundTripper that will sign requests
// using Amazon's Signature Verification V4 signing procedure. The request will
// then be handed off to the next RoundTripper provided by next. If next is nil,
// http.DefaultTransport will be used.
//
// Credentials for signing are retrieved using the the default AWS credential
// chain. If credentials cannot be found, an error will be returned.
func NewSigV4RoundTripper(cfg *Config, next http.RoundTripper) (http.RoundTripper, error) {
	ctx := context.Background()
	if next == nil {
		next = http.DefaultTransport
	}

	awsConfig := []func(*config.LoadOptions) error{}

	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		awsConfig = append(awsConfig, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, string(cfg.SecretKey), ""),
		))
	}

	if cfg.UseFIPSSTSEndpoint {
		awsConfig = append(awsConfig, config.WithUseFIPSEndpoint(aws.FIPSEndpointStateEnabled))
	} else {
		awsConfig = append(awsConfig, config.WithUseFIPSEndpoint(aws.FIPSEndpointStateDisabled))
	}

	if cfg.Region != "" {
		awsConfig = append(awsConfig, config.WithRegion(cfg.Region))
	}

	if cfg.Profile != "" {
		awsConfig = append(awsConfig, config.WithSharedConfigProfile(cfg.Profile))
	}

	awscfg, err := config.LoadDefaultConfig(
		ctx,
		awsConfig...,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create new AWS session: %w", err)
	}

	if _, err := awscfg.Credentials.Retrieve(ctx); err != nil {
		return nil, fmt.Errorf("could not get SigV4 credentials: %w", err)
	}

	if awscfg.Region == "" {
		return nil, fmt.Errorf("region not configured in sigv4 or in default credentials chain")
	}

	if cfg.RoleARN != "" {
		awscfg.Credentials = stscreds.NewAssumeRoleProvider(
			sts.NewFromConfig(awscfg),
			cfg.RoleARN,
			func(o *stscreds.AssumeRoleOptions) {
				if cfg.ExternalID != "" {
					o.ExternalID = aws.String(cfg.ExternalID)
				}
			},
		)
	}

	serviceName := "aps"

	if cfg.ServiceName != "" {
		serviceName = cfg.ServiceName
	}

	rt := &sigV4RoundTripper{
		region:      awscfg.Region,
		next:        next,
		creds:       aws.NewCredentialsCache(awscfg.Credentials, credentialCacheOptions),
		signer:      signer.NewSigner(),
		serviceName: serviceName,
	}
	rt.pool.New = rt.newBuf
	return rt, nil
}

func (rt *sigV4RoundTripper) newBuf() any {
	return bytes.NewBuffer(make([]byte, 0, 65536))
}

func (rt *sigV4RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// RoundTrippers must not modify the caller's request beyond consuming its
	// body, so clone first and perform every mutation on the clone.
	signReq := req.Clone(req.Context())
	for _, header := range sigv4HeaderDenylist {
		signReq.Header.Del(header)
	}

	buf := rt.pool.Get().(*bytes.Buffer)

	strHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // sha256 of an empty file

	defer func() {
		buf.Reset()
		rt.pool.Put(buf)
	}()

	if signReq.Body != nil {
		if _, err := io.Copy(buf, signReq.Body); err != nil {
			return nil, err
		}
		// Close the original body since we don't need it anymore.
		_ = signReq.Body.Close()

		// Ensure our seeker is back at the start of the buffer once we return.
		// Empty body is a valid situation
		seeker := bytes.NewReader(buf.Bytes())
		defer func() {
			_, _ = seeker.Seek(0, io.SeekStart)
		}()

		signReq.Body = io.NopCloser(seeker)
		hash := sha256.Sum256(buf.Bytes())
		strHash = hex.EncodeToString(hash[:])
	}

	// Declare the payload hash in the request itself, like the AWS SDKs do for
	// S3. Verifiers that never see the body (central STS validation) require
	// this header to reconstruct the canonical request.
	signReq.Header.Set("X-Amz-Content-Sha256", strHash)

	ctx := signReq.Context()
	creds, err := rt.creds.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("error retrieving credentials: %w", err)
	}

	err = rt.signer.SignHTTP(
		ctx,
		creds,
		signReq,
		strHash,
		rt.serviceName,
		rt.region,
		time.Now().UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// Set unsigned headers into the new req
	for _, header := range sigv4HeaderDenylist {
		headerValue := req.Header.Get(header)
		if headerValue != "" {
			signReq.Header.Set(header, headerValue)
		}
	}

	return rt.next.RoundTrip(signReq)
}

func credentialCacheOptions(options *aws.CredentialsCacheOptions) {
	options.ExpiryWindow = 30 * time.Second
	options.ExpiryWindowJitterFrac = 0.5
}
