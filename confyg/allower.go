package confyg

// Allower defines if a given verb and block combination is valid for
// configuration parsing.
//
// Return false if this verb and block pair is invalid.
type Allower interface {
	Allow(verb string, block bool) bool
}

// AllowerFunc implements Allower for inline definitions.
type AllowerFunc func(verb string, block bool) bool

// Allow implements Allower.
func (a AllowerFunc) Allow(verb string, block bool) bool {
	return a(verb, block)
}
