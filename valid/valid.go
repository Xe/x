// Package valid contains the Valider interface.
//
// The Valider interface is used to verify the "validity" of a value, where "validity" is defined by the implementing type.
//
// All examples are assuming the following struct:
//
//	type Example struct {
//	    Name string
//	}
//
// Where possible, implementors should use errors.Join to allow for multiple validation errors to be returned:
//
//	func (e *Example) Valid() error {
//	    var errs []error
//	    if e.Name == "" {
//	        errs = append(errs, errors.New("name is empty"))
//	    }
//	    if len(errs) == 0 {
//	        return nil
//	    }
//	    return errors.Join(errs...)
//	}
package valid

// Interface is an interface for types that can be validated.
type Interface interface {
	// Valid returns any validation errors for the value. If the value is valid, nil is returned.
	//
	// Where possible, implementors should use errors.Join to allow for multiple validation errors to be returned.
	Valid() error
}
