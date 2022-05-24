package httpserver

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/net/http2"

	"go.pact.im/x/process"
	"go.pact.im/x/third_party/github.com/lucas-clemente/quic-go/http3"
)

// Server is an HTTP server abstraction that supports H1, H2C, H2 and H3 with
// TLS.
type Server struct {
	log *zap.Logger

	handler http.Handler

	h1 *H1
	h2 *H2
	h3 *H3

	tcp []StreamSocket
	udp []PacketSocket

	connWG sync.WaitGroup
}

// NewServer returns a new Server instance with the given options.
func NewServer(o Options) *Server {
	o.setDefaults()

	return &Server{
		log:     o.Logger,
		handler: o.Handler,
		h1:      o.H1,
		h2:      o.H2,
		h3:      o.H3,
		tcp:     o.StreamSockets,
		udp:     o.PacketSockets,
	}
}

// Run runs the server. It guarantees that, on return, all ongoing requests are
// complete and underlying handler will not be called. The given callback is
// called after the server is initialized and is ready to accept requests.
func (s *Server) Run(ctx context.Context, callback func(ctx context.Context) error) error {
	handler := s.handler

	// HTTP/2 server runs handler in background goroutine. As a workaround,
	// once we know that all accepted connections are closed, we use Stop
	// method to abort future ServeHTTP invocations and wait for ongoing
	// handlers to return.
	//
	// See also https://go.dev/issue/37920
	sh := newAbortableHandler(handler)
	handler = sh

	// We want panics that are not http.ErrAbortHandler to be fatal.
	// See exitOnPanicHandler function documentation for motivation.
	handler = exitOnPanicHandler(handler)

	h1 := &http.Server{
		Handler:           handler,
		ReadTimeout:       s.h1.ReadTimeout,
		ReadHeaderTimeout: s.h1.ReadHeaderTimeout,
		WriteTimeout:      s.h1.WriteTimeout,
		IdleTimeout:       s.h1.IdleTimeout,
		MaxHeaderBytes:    s.h1.MaxHeaderBytes,
		ConnState:         connStateHook(s.connTrack, s.h1.ConnState),
		ErrorLog:          zap.NewStdLog(s.log),
		BaseContext:       s.h1.BaseContext,
		ConnContext:       s.h1.ConnContext,
	}
	if h1.BaseContext == nil {
		h1.BaseContext = func(net.Listener) context.Context {
			return ctx
		}
	}

	h2 := &http2.Server{
		MaxHandlers:                  s.h2.MaxHandlers,
		MaxConcurrentStreams:         s.h2.MaxConcurrentStreams,
		MaxReadFrameSize:             s.h2.MaxReadFrameSize,
		PermitProhibitedCipherSuites: s.h2.PermitProhibitedCipherSuites,
		IdleTimeout:                  s.h2.IdleTimeout,
		MaxUploadBufferPerConnection: s.h2.MaxUploadBufferPerConnection,
		MaxUploadBufferPerStream:     s.h2.MaxUploadBufferPerStream,
		NewWriteScheduler:            s.h2.NewWriteScheduler,
		CountError:                   s.h2.CountError,
	}

	// We always ConfigureServer sets
	if err := http2.ConfigureServer(h1, h2); err != nil {
		return err
	}
	// Revert unwanted side effects of http2.ConfigureServer.
	h1.TLSConfig = nil
	h2.IdleTimeout = s.h2.IdleTimeout

	var h2c *h2cHandler
	if s.h2.Cleartext {
		h2c = newH2CHandler(h1.Handler, h2, h1, s.log)
		h1.Handler = h2c
	}

	if m := s.h1.TLSNextProto; m != nil {
		mm := make(map[string]func(*http.Server, *tls.Conn, http.Handler), len(m))
		for k, v := range m {
			mm[k] = v
		}
		h1.TLSNextProto = mm
	}

	h3 := &http3.Server{
		Handler:            handler,
		MaxHeaderBytes:     s.h3.MaxHeaderBytes,
		EnableDatagrams:    s.h3.EnableDatagrams,
		AdditionalSettings: s.h3.AdditionalSettings,
		StreamHijacker:     s.h3.StreamHijacker,
		UniStreamHijacker:  s.h3.UniStreamHijacker,
	}

	var shutdownH1H2, shutdownH3 *sync.Once
	if len(s.tcp) > 0 {
		shutdownH1H2 = new(sync.Once)
	}
	if len(s.udp) > 0 {
		shutdownH3 = new(sync.Once)
	}

	n := len(s.tcp) + len(s.udp)
	procs := make([]process.Runnable, 0, n)
	for _, lc := range s.tcp {
		procs = append(procs, &serveH1H2{
			l: lc,
			h: h1,
			d: shutdownH1H2,
		})
	}
	for _, lc := range s.udp {
		procs = append(procs, &serveH3{
			l: lc,
			h: h3,
			s: s,
			d: shutdownH3,
		})
	}

	listenAndServe := process.Parallel(procs...)
	runError := listenAndServe.Run(ctx, callback)

	s.connWG.Wait()
	if s.h2.Cleartext {
		h2c.Shutdown(ctx)
	}
	sh.Stop()

	return runError
}
