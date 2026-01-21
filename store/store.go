package store

import (
	"context"
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ErrNotFound is returned when the store implementation cannot find the value
	// for a given key.
	ErrNotFound = errors.New("store: key not found")

	// ErrCantDecode is returned when a store adaptor cannot decode the store format
	// to a value used by the code.
	ErrCantDecode = errors.New("store: can't decode value")

	// ErrCantEncode is returned when a store adaptor cannot encode the value into
	// the format that the store uses.
	ErrCantEncode = errors.New("store: can't encode value")

	// ErrBadConfig is returned when a store adaptor's configuration is invalid.
	ErrBadConfig = errors.New("store: configuration is invalid")

	iopsMetrics = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "within_website_x",
		Subsystem: "store",
		Name:      "iops",
		Help:      "The number of times each store iop is called",
	}, []string{"driver", "action"})
)

// Interface defines the calls that Anubis uses for storage in a local or remote
// datastore. This can be implemented with an in-memory, on-disk, or in-database
// storage backend.
type Interface interface {
	// Delete removes a value from the store by key.
	Delete(ctx context.Context, key string) error

	// Exists returns nil if the key exists, ErrNotFound if it does not exist.
	Exists(ctx context.Context, key string) error

	// Get returns the value of a key assuming that value exists and has not expired.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set puts a value into the store that expires according to its expiry.
	Set(ctx context.Context, key string, value []byte) error

	// List lists the keys in this keyspace optionally matching by a prefix.
	List(ctx context.Context, prefix string) ([]string, error)
}
