package httpserver

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"sync"

	"go.uber.org/multierr"

	"go.pact.im/x/third_party/github.com/lucas-clemente/quic-go"
	"go.pact.im/x/third_party/github.com/lucas-clemente/quic-go/http3"
)

type serveH3 struct {
	l PacketSocket
	h *http3.Server
	s *Server
	d *sync.Once
}

func (s *serveH3) Run(ctx context.Context, callback func(ctx context.Context) error) error {
	pc, err := s.l.Listen(ctx)
	if err != nil {
		return err
	}

	c := s.s.h3

	tlsConf := c.TLSConfig
	if tlsConf == nil {
		tlsConf = &tls.Config{}
	}
	baseConf := http3.ConfigureTLSConfig(tlsConf) // does not modify tlsConf
	quicConf := c.QUICConfig
	if c.EnableDatagrams {
		if quicConf != nil {
			quicConf = quicConf.Clone()
		} else {
			quicConf = &quic.Config{}
		}
		quicConf.EnableDatagrams = true
	}

	// NB QUIC listener does not close the underlying PacketConn.
	ln, err := quic.ListenEarly(pc, baseConf, quicConf)
	if err != nil {
		return err
	}

	l := &onceCloseEarlyListener{EarlyListener: ln}

	fgctx, cancel := context.WithCancel(ctx)

	errc := make(chan error, 1)
	go func() {
		err := s.h.ServeListener(l)
		cancel()
		errc <- err
	}()

	callbackError := callback(fgctx)

	// Suppress duplicate shutdown calls; see also h1h2.go.
	s.d.Do(func() {
		_ = s.h.Shutdown(ctx)
	})

	closeError := l.Close()
	serveError := <-errc
	if errors.Is(serveError, quic.ErrServerClosed) || errors.Is(serveError, http.ErrServerClosed) {
		serveError = nil
	}
	closePacketConnError := pc.Close()

	return multierr.Combine(callbackError, serveError, closeError, closePacketConnError)
}

type onceCloseEarlyListener struct {
	quic.EarlyListener
	once sync.Once
	err  error
}

func (l *onceCloseEarlyListener) Close() error {
	l.once.Do(func() { l.err = l.EarlyListener.Close() })
	return l.err
}
