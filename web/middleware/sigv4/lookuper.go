package sigv4

import "context"

// Lookuper encapsulates the secret access key lookup so that the underlying
// logic can have arbitrary implementations.
type Lookuper interface {
	// Lookup resolves an access key id to its secret access key. Return
	// ErrUnknownKey for unknown keys. This is the one piece you must supply.
	Lookup(accessKeyID string) (secretAccessKey string, err error)
}

// LookuperFunc adapts an ordinary function to the Lookuper interface, the same
// way http.HandlerFunc adapts a function to http.Handler.
type LookuperFunc func(accessKeyID string) (secretAccessKey string, err error)

// Lookup calls f(accessKeyID).
func (f LookuperFunc) Lookup(accessKeyID string) (secretAccessKey string, err error) {
	return f(accessKeyID)
}

// SigningKeyLookuper resolves a credential scope to the SigV4 derived signing
// key HMAC(HMAC(HMAC(HMAC("AWS4"+secret, date), region), service),
// "aws4_request"). The arguments are the literal strings from the request's
// Credential= component, unnormalized. Return ErrUnknownKey when the key does
// not exist or may not sign (disabled key or user); any other error is treated
// as a server fault.
type SigningKeyLookuper interface {
	LookupSigningKey(ctx context.Context, accessKeyID, date, region, service string) ([]byte, error)
}

// SigningKeyLookuperFunc adapts an ordinary function to SigningKeyLookuper.
type SigningKeyLookuperFunc func(ctx context.Context, accessKeyID, date, region, service string) ([]byte, error)

// LookupSigningKey calls f.
func (f SigningKeyLookuperFunc) LookupSigningKey(ctx context.Context, accessKeyID, date, region, service string) ([]byte, error) {
	return f(ctx, accessKeyID, date, region, service)
}
