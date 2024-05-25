// Package tigris contains a Tigris client and helpers for interacting with Tigris.
//
// Tigris is a cloud storage service that provides a simple, scalable, and secure object storage solution. It is based on the S3 API, but has additional features that need these helpers.
package tigris

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/transport/http"
)

// WithHeader sets an arbitrary HTTP header on the request.
func WithHeader(key, value string) func(*s3.Options) {
	return func(options *s3.Options) {
		options.APIOptions = append(options.APIOptions, http.AddHeaderValue(key, value))
	}
}

// Region is a Tigris region from the documentation.
//
// https://www.tigrisdata.com/docs/concepts/regions/
type Region string

// Possible Tigris regions.
const (
	FRA Region = "fra" // Frankfurt, Germany
	GRU Region = "gru" // São Paulo, Brazil
	HKG Region = "hkg" // Hong Kong, China
	IAD Region = "iad" // Ashburn, Virginia, USA
	JNB Region = "jnb" // Johannesburg, South Africa
	LHR Region = "lhr" // London, UK
	MAD Region = "mad" // Madrid, Spain
	NRT Region = "nrt" // Tokyo (Narita), Japan
	ORD Region = "ord" // Chicago, Illinois, USA
	SIN Region = "sin" // Singapore
	SJC Region = "sjc" // San Jose, California, USA
	SYD Region = "syd" // Sydney, Australia
)

// WithStaticReplicationRegions sets the regions where the object will be replicated.
//
// Note that this will cause you to be charged multiple times for the same object, once per region.
func WithStaticReplicationRegions(regions []Region) func(*s3.Options) {
	regionsString := make([]string, 0, len(regions))
	for _, r := range regions {
		regionsString = append(regionsString, string(r))
	}

	return WithHeader("X-Tigris-Regions", strings.Join(regionsString, ","))
}

// WithQuery lets you filter objects in a ListObjectsV2 request.
//
// This functions like the WHERE clause in SQL, but for S3 objects. For more information, see the Tigris documentation[1].
//
// [1]: https://www.tigrisdata.com/docs/objects/query-metadata/
func WithQuery(query string) func(*s3.Options) {
	return WithHeader("X-Tigris-Query", query)
}

// WithCreateIfNotExists will create the object if it doesn't exist.
//
// See the Tigris documentation[1] for more information.
//
// [1]: https://www.tigrisdata.com/docs/objects/conditionals/
func WithCreateObjectIfNotExists() func(*s3.Options) {
	return WithHeader("If-Match", `""`)
}

// WithIfEtagMatches sets the ETag that the object must match.
//
// See the Tigris documentation[1] for more information.
//
// [1]: https://www.tigrisdata.com/docs/objects/conditionals/
func WithIfEtagMatches(etag string) func(*s3.Options) {
	return WithHeader("If-Match", etag)
}

// WithModifiedSince lets you proceed with operation if object was modified after provided date (RFC1123).
//
// See the Tigris documentation[1] for more information.
//
// [1]: https://www.tigrisdata.com/docs/objects/conditionals/
func WithModifiedSince(modifiedSince time.Time) func(*s3.Options) {
	return WithHeader("If-Modified-Since", modifiedSince.Format(time.RFC1123))
}

// WithUnmodifiedSince lets you proceed with operation if object was not modified after provided date (RFC1123).
//
// See the Tigris documentation[1] for more information.
//
// [1]: https://www.tigrisdata.com/docs/objects/conditionals/
func WithUnmodifiedSince(unmodifiedSince time.Time) func(*s3.Options) {
	return WithHeader("If-Unmodified-Since", unmodifiedSince.Format(time.RFC1123))
}

// WithCompareAndSwap tells Tigris to skip the cache and read the object from its designated region.
//
// This is only used on GET requests.
//
// See the Tigris documentation[1] for more information.
//
// [1]: https://www.tigrisdata.com/docs/objects/conditionals/
func WithCompareAndSwap() func(*s3.Options) {
	return WithHeader("X-Tigris-CAS", "true")
}

// Client returns a new S3 client wired up for Tigris.
func Client(ctx context.Context) (*s3.Client, error) {
	cfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load Tigris config: %w", err)
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("https://fly.storage.tigris.dev")
		o.Region = "auto"
	}), nil
}
