// Command friendship-ended makes "Friendship ended with X, now Y is my best
// friend" images and stores them in Tigris.
//
// It is a port of https://github.com/lmaucoin/friendship-ended by Leigh
// Aucoin.
//
// Credentials come from the standard AWS environment variables (for
// example AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY, or AWS_PROFILE). For
// local development, put AWS_PROFILE=tigris in cmd/friendship-ended/.env
// and run the binary from that directory: internal.HandleStartup() loads
// .env from the current working directory, not from the source tree.
package main

//go:generate go tool templ generate
