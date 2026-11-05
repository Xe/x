// Package examples contains the source documents for the IAM CLI's built-in
// policy templates. Editing an example updates its template on the next build.
package examples

import "embed"

// Policies contains the example policy documents, keyed by filename.
//
//go:embed *.json
var Policies embed.FS
