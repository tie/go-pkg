package httpserver

import (
	"bufio"
	"context"
	"crypto/tls"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/net/http2"
)

type tlsListener struct {
	Socket    StreamSocket
	TLSConfig *tls.Config
	Optional  bool
	Timeout   time.Duration
}

// TLS returns a StreamSocket for the given TCP address that adds TLS layer
// to the accepted connections.
//
// If NextProtos is not set, it defaults to h2 to advertise HTTP/2 support.
func TLS(address string, c *tls.Config) StreamSocket {
	return &tlsListener{
		Socket:    TCP(address),
		TLSConfig: defaultNextProtosH2(c),
	}
}

// OptionalTLS returns a StreamSocket for the given TCP address that adds TLS
// layer to the accepted connections if the first sniffed byte is the handshake
// record type.
//
// If NextProtos is not set, it defaults to h2 to advertise HTTP/2 support.
//
// The timeout is used to set read deadline for the first byte. If not set, it
// defaults to 50ms.
//
// Note that, since Accept is usually called in a loop, OptionalTLS may cause
// DoS if used incorrectly. In particular, each accepted connection will block
// for at most timeout duration to decide whether TLS should be enabled. The
// worst scenario is when the server is under a storm of connections that do
// not send any data, slowing down the overall rate of accepted connections and
// increasing latency. To avoid this issue, never use addresses served using
// OptionalTLS for primary API endpoints.
//
// This function is intended for transitioning internal services from HTTP-based
// authentication (e.g. HTTP Authorization header) to mutual TLS authentication.
func OptionalTLS(address string, c *tls.Config, timeout time.Duration) StreamSocket {
	if timeout <= 0 {
		timeout = 50 * time.Millisecond
	}
	return &tlsListener{
		Socket:    TCP(address),
		TLSConfig: defaultNextProtosH2(c),
		Optional:  true,
		Timeout:   timeout,
	}
}

func (l *tlsListener) Listen(ctx context.Context) (net.Listener, error) {
	ln, err := l.Socket.Listen(ctx)
	if err != nil {
		return nil, err
	}
	if !l.Optional {
		return tls.NewListener(ln, l.TLSConfig), nil
	}
	return &tlsAcceptor{
		Listener:  ln,
		TLSConfig: l.TLSConfig,
		Timeout:   l.Timeout,
	}, nil
}

type tlsAcceptor struct {
	net.Listener
	TLSConfig *tls.Config
	Timeout   time.Duration
}

func (t *tlsAcceptor) Accept() (net.Conn, error) {
	c, err := t.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return t.sniff(c), nil
}

// sniff sniffs the first byte of the connection
func (t *tlsAcceptor) sniff(c net.Conn) net.Conn {
	const recordTypeHandshake = 22

	br := bufio.NewReaderSize(c, 1)

	bc := &bufferedConn{
		Conn: c,
		br:   br,
	}

	_ = c.SetReadDeadline(time.Now().Add(t.Timeout))
	buf, err := br.Peek(1)
	_ = c.SetReadDeadline(time.Time{})

	if err != nil || len(buf) < 1 {
		return bc
	}
	if buf[0] != recordTypeHandshake {
		return bc
	}
	return tls.Server(bc, t.TLSConfig)
}

type bufferedConn struct {
	net.Conn
	br *bufio.Reader
}

func (b *bufferedConn) Read(p []byte) (int, error) {
	return b.br.Read(p)
}

// ReadFrom implements the io.ReadFrom interface. It is needed since net/http
// uses this method for writing responses using system calls.
//
// See https://github.com/golang/go/blob/fd6c556dc82253722a7f7b9f554a1892b0ede36e/src/net/http/server.go#L565-L568
func (b *bufferedConn) ReadFrom(r io.Reader) (int64, error) {
	rf, ok := b.Conn.(io.ReaderFrom)
	if !ok {
		return fallbackReadFrom(b.Conn, r)
	}
	return rf.ReadFrom(r)
}

// copyBufPool is copied from net/http for fallback ReadFrom implementation.
//
// See https://github.com/golang/go/blob/fd6c556dc82253722a7f7b9f554a1892b0ede36e/src/net/http/server.go#L801-L806
var copyBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 32*1024)
		return &b
	},
}

// fallbackReadFrom is a fallback implementation of the io.ReaderFrom interface
// for bufferedConn if the underlying net.Conn does not implement it.
func fallbackReadFrom(w io.Writer, r io.Reader) (n int64, err error) {
	bufp := copyBufPool.Get().(*[]byte)
	buf := *bufp
	defer copyBufPool.Put(bufp)

	type writerOnly struct {
		io.Writer
	}
	return io.CopyBuffer(writerOnly{w}, r, buf)
}

// defaultNextProtosH2 returns a TLS configuration with NextProtos defaulting
// to single "h2" element.
func defaultNextProtosH2(c *tls.Config) *tls.Config {
	if c.NextProtos != nil {
		return c
	}
	c = c.Clone()
	c.NextProtos = []string{http2.NextProtoTLS}
	return c
}
