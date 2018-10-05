// Package localca uses an autocert.Cache to store and generate TLS certificates
// for domains on demand.
//
// This is kind of powerful, and as such it is limited to only generate
// certificates as subdomains of a given domain.
//
// The design and implementation of this is kinda stolen from minica[1].
//
// [1]: https://github.com/jsha/minica
package localca
