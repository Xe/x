package sigv4

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
