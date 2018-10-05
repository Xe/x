# madon

Golang library for the Mastodon API

[![godoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/McKael/madon)
[![license](https://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/McKael/madon/master/LICENSE)
[![build](https://travis-ci.org/McKael/madon.svg?branch=master)](https://travis-ci.org/McKael/madon)
[![Go Report Card](https://goreportcard.com/badge/github.com/McKael/madon)](https://goreportcard.com/report/github.com/McKael/madon)

`madon` is a [Go](https://golang.org/) library to access the Mastondon REST API.

This implementation covers 100% of the current API, including the streaming API.

The [madonctl](https://github.com/McKael/madonctl) console client uses this library exhaustively.

## Installation

To install the library (Go >= v1.5 required):

    go get github.com/McKael/madon

You can test it with my CLI tool:

    go get github.com/McKael/madonctl

## Usage

This section has not been written yet (PR welcome).

For now please check [godoc](https://godoc.org/github.com/McKael/madon) and
check the [madonctl](https://github.com/McKael/madonctl) project
implementation.

## History

This API implementation was initially submitted as a PR for gondole.

The repository is actually a fork of my gondole branch so that
history and credits are preserved.

## References

- [madonctl](https://github.com/McKael/madonctl) (console client based on madon)
- [Mastodon API documentation](https://github.com/tootsuite/documentation/blob/master/Using-the-API/API.md)
- [Mastodon Streaming API documentation](https://github.com/tootsuite/documentation/blob/master/Using-the-API/Streaming-API.md)
- [Mastodon repository](https://github.com/tootsuite/mastodon)
