package httpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

// This file contains partial HTTP/2 implementation for cleartext connections.
//
// Prior Knowledge and Upgrade
//
// As per RFC 7540 Section 3.2, “requests that contain a payload body MUST be
// sent in their entirety before the client can send HTTP/2 frames.” If the
// server supports HTTP/2, it “accepts the upgrade with a 101 (Switching
// Protocols) response.” “After commencing the HTTP/2 connection, stream 1 is
// used for the response.”
//
// To make the implementation easier and avoid the issue altogether, we do not
// support connection upgrades. That is, only requests made with HTTP/2 prior
// knowledge are valid for H2C.
//
// See also https://go.dev/issue/38064
//
// Graceful Shutdown
//
// There is a race between http.Server’s Shutdown and http2.Server’s ServeConn
// method for connections served using golang.org/x/net/http2/h2c hanlder that
// seems to be undocumented (we couldn’t find find any relevant issues).
//
// The http2 package handles graceful shutdown by registering OnShutdown hook
// in ConfigureServer. The hook starts graceful shutdown for all connections
// on http2.Server. When the Shutdown is called, it runs all registered hooks.
//
// When connection receives an H2C request, it is hijacked (i.e. removed from
// connections tracked by http.Server) and passed to http2.Server’s ServeConn.
// If a shutdown happens between these two events, the shutdown hook will not
// trigger sending GOAWAY and connection will not be terminated by the server.
//
// See also https://go.dev/issue/26682
//
// We handle this issue by tracking connection hijacking (see track.go) and
// calling http.Server’s Shutdown method again once connections are either
// closed or hijacked.
//
// Consider the following order of events:
//
//   1. (*http.Server).Serve sets connection state to New.
//   2. ConnState hook increments Server’s connWG.
//   2. A new serve goroutine is created for the connection.
//   4. ServeHTTP is called for a request that is an h2c upgrade.
//   5. We increment handler’s trackWG and serveWG.
//   6. Handler hijacks the connection or decrements trackWG/serveWG on error.
//   7. Connection state is set to Hijacked thus decrementing Server’s connWG.
//   8. Handler runs http2.Server’s ServeConn for the connection.
//   9. Server begins shutdown process.
//  10. http.Server’s Serve, Close and Shutdown methods return.
//  11. We wait on connWG for connections to be either closed or hijacked.
//
// At this point, serveWG tracks hijacked connections that have active ServeConn
// calls. Since http.Server’s Serve already returned and we’ve waited on connWG,
// there should be no more ServeHTTP calls on the handler. That is, it is safe
// to wait on trackWG and serveWG.

// h2cHandler is a Handler which implements h2c by hijacking the HTTP/1 traffic
// that should be h2c traffic.
type h2cHandler struct {
	handler    http.Handler
	fallback   http.Handler
	baseConfig *http.Server
	h1         *http.Server
	h2         *http2.Server
	log        *zap.Logger

	trackWG sync.WaitGroup
	serveWG sync.WaitGroup

	connsMu sync.Mutex
	conns   map[net.Conn]bool
}

// newH2CHandler returns a new h2cHandler instance.
func newH2CHandler(h http.Handler, h2 *http2.Server, h1 *http.Server, l *zap.Logger) *h2cHandler {
	// Clone h1 for BaseConfig, otherwise HTTP/2 server calls ConnState
	// hook that we use to track HTTP/1 connections.
	baseConfig := &http.Server{
		ReadTimeout:    h1.ReadTimeout,
		WriteTimeout:   h1.WriteTimeout,
		MaxHeaderBytes: h1.MaxHeaderBytes,
		ErrorLog:       h1.ErrorLog,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http2’s ServeConn sets associated server to BaseConfig. This
		// is not true in our case, so override it to h1.
		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), http.ServerContextKey, h1)))
	})

	h2c := &h2cHandler{
		handler:    handler,
		fallback:   h,
		baseConfig: baseConfig,
		h1:         h1,
		h2:         h2,
		log:        l,
	}

	// ServeConn either returns on early error or updates the connection
	// state. We use this behavior to wait until the connection is tracked
	// in underlying http2.Server state so that http.Server’s OnShutdown
	// hook can start graceful shutdown.
	h2c.baseConfig.ConnState = func(c net.Conn, _ http.ConnState) {
		h2c.setTracked(c)
	}
	return h2c
}

// addPending adds the connection to the set of pending connections.
func (h *h2cHandler) addPending(c net.Conn) {
	h.connsMu.Lock()
	defer h.connsMu.Unlock()

	if h.conns == nil {
		h.conns = make(map[net.Conn]bool)
	}

	h.conns[c] = false
	h.trackWG.Add(1)
	h.serveWG.Add(1)
}

// setTracked signals that the given pending connection is now tracked by the
// http2.Server.
func (h *h2cHandler) setTracked(c net.Conn) {
	h.connsMu.Lock()
	defer h.connsMu.Unlock()

	tracked, exists := h.conns[c]
	if !exists || tracked {
		return
	}
	h.conns[c] = true
	h.trackWG.Done()
}

// removeConn removes the given connection from the pending connections set.
func (h *h2cHandler) removeConn(c net.Conn) {
	h.connsMu.Lock()
	defer h.connsMu.Unlock()

	tracked, exists := h.conns[c]
	if !exists {
		return
	}
	delete(h.conns, c)
	if len(h.conns) == 0 {
		h.conns = nil
	}

	// Decrement trackWG if setTracked has not been called for c.
	if !tracked {
		h.trackWG.Done()
	}
	h.serveWG.Done()
}

// Shutdown waits for pending connections to become tracked by the underlying
// http2.Server and starts graceful shutdown. It must be called after the
// http.Server was shut down and accepted connections were either hijacked or
// closed (see ConnState hook and also track.go).
//
// If the context expires, pending connections are forcefully closed.
func (h *h2cHandler) Shutdown(ctx context.Context) {
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-done:
			return
		case <-ctx.Done():
		}
		h.closeConns()
	}()

	h.trackWG.Wait()       // wait until connections are tracked
	_ = h.h1.Shutdown(ctx) // run OnShutdown hooks
	h.serveWG.Wait()       // wait for shutdown
	close(done)
	wg.Wait()
}

// closeConns forcefully closes all underlying connections.
func (h *h2cHandler) closeConns() {
	h.connsMu.Lock()
	defer h.connsMu.Unlock()
	for c := range h.conns {
		_ = c.Close()
	}
}

// ServeHTTP implements the http.Handler interface.
func (h *h2cHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle h2c with prior knowledge (RFC 7540 Section 3.4)
	if r.Method == "PRI" && len(r.Header) == 0 && r.URL.Path == "*" && r.Proto == "HTTP/2.0" {
		h.serveH2CWithPriorKnowledge(w, r)
		return
	}
	h.fallback.ServeHTTP(w, r)
}

// serveH2CWithPriorKnowledge initializes H2C and passes the hijacked connection
// to HTTP/2 server. If H2C initialization fails, it aborts the handler.
func (h *h2cHandler) serveH2CWithPriorKnowledge(w http.ResponseWriter, r *http.Request) {
	conn, err := h.initH2CWithPriorKnowledge(w)
	if err != nil {
		h.log.Info("Failed to initialize H2C with prior knowledge",
			zap.Error(err),
		)
		panic(http.ErrAbortHandler)
	}
	defer h.removeConn(conn)
	h.h2.ServeConn(conn, &http2.ServeConnOpts{
		Context:    r.Context(),
		Handler:    h.handler,
		BaseConfig: h.baseConfig,
	})
}

// initH2CWithPriorKnowledge implements creating a h2c connection with prior
// knowledge (Section 3.4) and creates a net.Conn suitable for http2.ServeConn.
// All we have to do is look for the client preface that is suppose to be part
// of the body, and reforward the client preface on the net.Conn this function
// creates.
func (h *h2cHandler) initH2CWithPriorKnowledge(w http.ResponseWriter) (_ net.Conn, err error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("hijack not supported")
	}

	// Create an unitialized connection and add it to the set of pending
	// connections.
	//
	// Note that Hijack removes the hijacked conn from http.Server’s tracked
	// connections set. If we call addPending after initializing connection,
	// there is a potential time frame when the connection is not tracked at
	// all and is therefore invisible to both graceful and forceful shutdown
	// processes.
	//
	c := &rwConn{}
	h.addPending(c)
	defer func() {
		if err != nil {
			return
		}
		h.removeConn(c)
	}()

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, fmt.Errorf("hijack failed: %w", err)
	}

	const expectedBody = "SM\r\n\r\n"

	buf := make([]byte, len(expectedBody))
	n, err := io.ReadFull(rw, buf)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("could not read from the buffer: %w", err)
	}

	if string(buf[:n]) != expectedBody {
		_ = conn.Close()
		return nil, errors.New("invalid client preface")
	}

	*c = rwConn{
		Conn:      conn,
		Reader:    io.MultiReader(strings.NewReader(http2.ClientPreface), rw),
		BufWriter: rw.Writer,
	}
	return c, nil
}

// rwConn implements net.Conn but overrides Read and Write so that reads and
// writes are forwarded to the provided io.Reader and bufWriter.
type rwConn struct {
	net.Conn
	io.Reader
	BufWriter bufWriter
}

// Read forwards reads to the underlying Reader.
func (c *rwConn) Read(p []byte) (int, error) {
	return c.Reader.Read(p)
}

// Write forwards writes to the underlying bufWriter and immediately flushes.
func (c *rwConn) Write(p []byte) (int, error) {
	n, err := c.BufWriter.Write(p)
	if err := c.BufWriter.Flush(); err != nil {
		return 0, err
	}
	return n, err
}

// bufWriter is a Writer interface that also has a Flush method.
type bufWriter interface {
	io.Writer
	Flush() error
}
