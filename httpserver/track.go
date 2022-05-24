package httpserver

import (
	"net"
	"net/http"
)

// This file contains the glue neccesary to make HTTP packages use goroutines
// that adhere to structured concurrency rules.
//
// In particular, we want the following guarantee: once Serve/ServeConn/Shutdown
// methods return, underlying handler’s ServeHTTP method must not be called.
//
// This is mostly true for HTTP/1 connections when using the net/http package,
// but we need to track new, hijacked and closed connections. See connTrack for
// more details.
//
// HTTP/2 Cleartext (H2C) is the same as HTTP/2 except that the connection is
// hijacked after an upgrade from HTTP/1. We can guarantee that ServeHTTP will
// not be called after shutdown for HTTP/1 (see above), so we track ServeHTTP
// and ServeConn calls and wait for running handlers after shutdown. See also
// h2c.go for details.
//
// For HTTP/2’s issue https://go.dev/issue/37920 we use handler that aborts
// requests once we know that all connections are closed. Otherwise HTTP/2
// tracking is similar to HTTP/1 (i.e. uses ConnState hooks) except that the
// former uses TLSNextProto callback instead of http.Handler directly.
//
// For HTTP/3 we vendor a modified version where we implement Shutdown method
// that waits for handlers to return so there is no need to track anything.

// connTrack is a server ConnState hook that tracks new, hijacked and closed
// connections to enforce the guarantee that http.Handler is not called after
// Run method returns.
func (s *Server) connTrack(_ net.Conn, state http.ConnState) {
	switch state {
	// StateNew is the initial state and is set before connection is served.
	// That is, this hook runs before serve goroutine is spawned for the
	// connection (in exported Serve method).
	//
	// We call s.connWG.Wait() after Serve returns to ensure that there are no
	// races, deadlocks and panics.
	case http.StateNew:
		s.connWG.Add(1)
	// If the connection was hijacked, handler is responsible for tracking
	// it once ServeHTTP returns. From our perspective, this is equivalent
	// to closing the connection. Otherwise we have returned from connection
	// serve goroutine and are done with this connection.
	//
	// Note that h2c handler hijacks the connection and runs http2 ServeConn
	// on connection serve goroutine. When a connection is hijacked for h2c,
	// we track it separately using h2cHandler.
	case http.StateHijacked, http.StateClosed:
		s.connWG.Done()
	}
}

// connStateHook combines two http.Server ConnState hooks.
func connStateHook(a, b func(net.Conn, http.ConnState)) func(net.Conn, http.ConnState) {
	switch {
	case a == nil && b == nil:
		return nil
	case a == nil:
		return b
	case b == nil:
		return a
	}
	return func(conn net.Conn, state http.ConnState) {
		a(conn, state)
		b(conn, state)
	}
}
