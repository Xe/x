// Package maybedoer contains a pipeline of actions that might fail. If any action
// in the chain fails, no further actions take place and the error becomes the pipeline
// error.
package maybedoer

// Impl sequences a set of actions to be performed via calls to
// `Maybe` such that any previous error prevents new actions from being
// performed.
//
// This is, conceptually, just a go-ification of the Maybe monad.
type Impl struct {
	err error
}

// Maybe performs `f` if no previous call to a Maybe'd action resulted
// in an error
func (c *Impl) Maybe(f func() error) {
	if c.err == nil {
		c.err = f()
	}
}

// Error returns the first error encountered in the Error chain.
func (c *Impl) Error() error {
	return c.err
}
