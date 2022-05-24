// Package httpserver provides HTTP/1, HTTP/2 (including H2C) and HTTP/3 server
// implementation with opportunistic TLS support. It also exposes convenience
// helpers for running gRPC services.
//
// The server implementation provides strong guarantees on handler lifecycle.
// In particular, handler will not be called after server shutdown. Note that
// this is not true out-of-the-box in case of golang.org/x/net/http2 package,
// and therefore Go’s net/http that bundles it for HTTP/2 support. See also
// https://go.dev/issue/37920.
//
// Warning: the API in this package is experimental until there is an official
// Go support for HTTP/3 and QUIC protocol. Currently a modified third-party
// implementation is used.
package httpserver
