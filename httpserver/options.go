package httpserver

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/net/http2"

	"go.pact.im/x/third_party/github.com/lucas-clemente/quic-go"
	"go.pact.im/x/third_party/github.com/lucas-clemente/quic-go/http3"
)

// H1 contains HTTP/1 server configuration.
type H1 struct {
	// ReadTimeout is the maximum duration for reading the entire request,
	// including the body. A zero or negative value means there will be no
	// timeout.
	//
	// Because ReadTimeout does not let Handlers make per-request decisions
	// on each request body’s acceptable deadline or upload rate, most users
	// will prefer to use ReadHeaderTimeout. It is valid to use them both.
	ReadTimeout time.Duration

	// ReadHeaderTimeout is the amount of time allowed to read request
	// headers. The connection’s read deadline is reset after reading the
	// headers and the Handler can decide what is considered too slow for
	// the body. If ReadHeaderTimeout is zero, the value of ReadTimeout is
	// used. If both are zero, there is no timeout.
	ReadHeaderTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out writes of the
	// response. It is reset whenever a new request’s header is read. Like
	// ReadTimeout, it does not let Handlers make decisions on a per-request
	// basis. A zero or negative value means there will be no timeout.
	WriteTimeout time.Duration

	// IdleTimeout is the maximum amount of time to wait for the next
	// request when keep-alives are enabled. If IdleTimeout is zero, the
	// value of ReadTimeout is used. If both are zero, there is no timeout.
	IdleTimeout time.Duration

	// MaxHeaderBytes controls the maximum number of bytes the server will
	// read parsing the request header’s keys and values, including the
	// request line. It does not limit the size of the request body. If
	// zero, http.DefaultMaxHeaderBytes is used.
	MaxHeaderBytes int

	// TLSNextProto optionally specifies a function to take over ownership
	// of the provided TLS connection when an ALPN protocol upgrade has
	// occurred. The map key is the protocol name negotiated. The Handler
	// argument should be used to handle HTTP requests and will initialize
	// the Request’s TLS and RemoteAddr if not already set. The connection
	// is automatically closed when the function returns.
	//
	// Note that golang.org/x/net/http2.NextProtoTLS ("h2" string) entry is
	// reserved for HTTP/2 server.
	TLSNextProto map[string]func(*http.Server, *tls.Conn, http.Handler)

	// ConnState specifies an optional callback function that is called when
	// a client connection changes state. See the http.ConnState type and
	// associated constants for details.
	ConnState func(net.Conn, http.ConnState)

	// BaseContext optionally specifies a function that returns the base
	// context for incoming requests on this server. The provided Listener
	// is the specific Listener that’s about to start accepting requests.
	// If BaseContext is nil, the default is the context passed to the Run.
	// method. If non-nil, it must return a non-nil context.
	BaseContext func(net.Listener) context.Context

	// ConnContext optionally specifies a function that modifies the context
	// used for a new connection c. The provided ctx is derived from the
	// base context and has a ServerContextKey value.
	ConnContext func(ctx context.Context, c net.Conn) context.Context
}

// H2 contains HTTP/2 server configuration.
type H2 struct {
	// Cleartext enables H2C (HTTP/2 Cleartext) support.
	Cleartext bool

	// MaxHandlers limits the number of http.Handler ServeHTTP goroutines
	// which may run at a time over all connections. A zero or negative
	// value means there will be no limit.
	//
	// Currently this option is not implemented in golang.org/x/net/http2
	// package that is used under-the-hood for H2 connections.
	MaxHandlers int

	// MaxConcurrentStreams optionally specifies the number of concurrent
	// streams that each client may have open at a time. This is unrelated
	// to the number of http.Handler goroutines which may be active
	// globally, which is MaxHandlers. If zero, MaxConcurrentStreams
	// defaults to at least 100, per the HTTP/2 spec’s recommendations.
	MaxConcurrentStreams uint32

	// MaxReadFrameSize optionally specifies the largest frame this server
	// is willing to read. A valid value is between 16k and 16M, inclusive.
	// If zero or otherwise invalid, a default value is used.
	MaxReadFrameSize uint32

	// PermitProhibitedCipherSuites, if true, permits the use of cipher
	// suites prohibited by the HTTP/2 spec.
	PermitProhibitedCipherSuites bool

	// IdleTimeout specifies how long until idle clients should be closed
	// with a GOAWAY frame. PING frames are not considered activity for the
	// purposes of IdleTimeout.
	IdleTimeout time.Duration

	// MaxUploadBufferPerConnection is the size of the initial flow control
	// window for each connections. The HTTP/2 spec does not allow this to
	// be smaller than 65535 or larger than 2^32-1. If the value is outside
	// this range, a default value will be used instead.
	MaxUploadBufferPerConnection int32

	// MaxUploadBufferPerStream is the size of the initial flow control
	// window for each stream. The HTTP/2 spec does not allow this to be
	// larger than 2^32-1. If the value is zero or larger than the maximum,
	// a default value will be used instead.
	MaxUploadBufferPerStream int32

	// NewWriteScheduler constructs a write scheduler for a connection. If
	// nil, a default scheduler is chosen.
	NewWriteScheduler func() http2.WriteScheduler

	// CountError, if non-nil, is called on HTTP/2 server errors. It’s
	// intended to increment a metric for monitoring, such as an expvar or
	// Prometheus metric.  The errType argument consists of only ASCII word
	// characters.
	CountError func(errType string)
}

// H3 contains HTTP/3 server configuration.
type H3 struct {
	// MaxHeaderBytes controls the maximum number of bytes the server will
	// read parsing the request header’s keys and values, including the
	// request line. It does not limit the size of the request body. If
	// zero, http.DefaultMaxHeaderBytes is used.
	MaxHeaderBytes int

	// TLSConfig provides TLS configuration for QUIC connections. It is
	// required for HTTP/3.
	TLSConfig *tls.Config

	// QUICConfig provides a way to set parameters of the QUIC connection.
	// If nil, it uses reasonable default values.
	QUICConfig *quic.Config

	// EnableDatagrams enables support for HTTP/3 datagrams. If set to true,
	// QUICConfig.EnableDatagram will be set.
	//
	// See https://datatracker.ietf.org/doc/html/draft-ietf-masque-h3-datagram-07.
	EnableDatagrams bool

	// AdditionalSettings contains additional HTTP/3 settings. It is invalid
	// to specify any settings defined by the HTTP/3 draft and the datagram
	// draft.
	AdditionalSettings map[uint64]uint64

	// StreamHijacker, if set, is called for the first unknown frame parsed
	// on a bidirectional stream. It is called right after parsing the frame
	// type. Callers can either process the frame and return control of the
	// stream back to HTTP/3 (by returning hijacked false). Alternatively,
	// callers can take over the QUIC stream (by returning hijacked true).
	StreamHijacker func(http3.FrameType, quic.Connection, quic.Stream) (hijacked bool, err error)

	// UniStreamHijacker, if set, is called for unknown unidirectional
	// stream of unknown stream type.
	UniStreamHijacker func(http3.StreamType, quic.Connection, quic.ReceiveStream) (hijacked bool)
}

// Options is a set of options for App constructor.
type Options struct {
	// Logger is a logger to use for server logs. If not set, logs are not
	// written.
	Logger *zap.Logger

	// Handler is a handler for HTTP requests. Defaults to http.NotFound.
	Handler http.Handler

	// H1 contains the base HTTP/1 server configuration. Defaults to server
	// with header read timeout and connection idle time set.
	//
	// See https://ieftimov.com/post/make-resilient-golang-net-http-servers-using-timeouts-deadlines-context-cancellation
	H1 *H1

	// H2 contains the base HTTP/2 server configuration. Defaults to server
	// with H2C enabled and connection idle time set.
	H2 *H2

	// H3 contains the base HTTP/3 server configuration.
	H3 *H3

	// StreamSockets specifies net.Listener sockets for server.
	StreamSockets []StreamSocket

	// PacketSockets specifies net.PacketConn sockets for server.
	PacketSockets []PacketSocket
}

// setDefaults sets default values for unspecified options.
func (o *Options) setDefaults() {
	if o.Logger == nil {
		o.Logger = zap.NewNop()
	}
	if o.Handler == nil {
		o.Handler = http.NotFoundHandler()
	}
	if o.H1 == nil {
		o.H1 = &H1{
			ReadHeaderTimeout: time.Second,
			IdleTimeout:       time.Minute,
		}
	}
	if o.H2 == nil {
		o.H2 = &H2{
			Cleartext:   true,
			IdleTimeout: time.Minute,
		}
	}
	if o.H3 == nil {
		o.H3 = &H3{}
	}
}
