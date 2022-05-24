package httpserver

import (
	"net/http"
	"strings"
)

// grpcMux is an http.Handler implementation that uses grpc.Server instance as
// an http.Handler to serve gRPC requests, and otherwise fallback to the default
// handler.
type grpcMux struct {
	h, g http.Handler
}

// GRPC returns http.Handler that uses g handler for gRPC requests and h
// otherwise.
func GRPC(h, g http.Handler) http.Handler {
	return &grpcMux{h, g}
}

// ServeHTTP implements the http.Handler interface.
func (m *grpcMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
		m.g.ServeHTTP(w, r)
	} else {
		m.h.ServeHTTP(w, r)
	}
}
