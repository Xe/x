package confyg

// Allower defines if a given verb and block combination is valid for
// configuration parsing.
//
// If this is intended to be a statement-like verb, block should be set
// to false. If this is intended to be a block-like verb, block should
// be set to true.
type Allower interface {
	Allow(verb string, block bool) bool
}

// AllowerFunc implements Allower for inline definitions.
type AllowerFunc func(verb string, block bool) bool

// Allow implements Allower.
func (a AllowerFunc) Allow(verb string, block bool) bool {
	return a(verb, block)
}
