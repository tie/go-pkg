package httpserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"

	"go.uber.org/multierr"
)

type serveH1H2 struct {
	l StreamSocket
	h *http.Server
	d *sync.Once
}

func (s *serveH1H2) Run(ctx context.Context, callback func(ctx context.Context) error) error {
	ln, err := s.l.Listen(ctx)
	if err != nil {
		return err
	}
	l := &onceCloseListener{Listener: ln}

	fgctx, cancel := context.WithCancel(ctx)

	errc := make(chan error, 1)
	go func() {
		err := s.h.Serve(l)
		cancel()
		errc <- err
	}()

	callbackError := callback(fgctx)

	// Suppress duplicate shutdown calls since they are idempotent.
	// Note that this will shutdown all serveH1H2 processes for the
	// underlying http.Server instance.
	s.d.Do(func() {
		s.gracefulShutdown(ctx)
	})

	// There is, however, a race between Serve and Close: the latter does
	// not wait for Serve to complete (like Shutdown does by polling the
	// number of active listeners) and immediately proceeds with closing
	// all active connections. If the shutdown is triggered after connection
	// was accepted but before its state was set (i.e. it is not tracked by
	// the server yet), it will not be forcefully closed by Close method.
	//
	// See https://go.dev/issue/48642
	//
	// At this point we know that Serve returned so all accepted connections
	// are tracked by the server and we are no longer eligible for graceful
	// shutdown, so forcefully close all remaining connections.
	_ = s.h.Close()

	closeError := l.Close()
	serveError := <-errc
	if errors.Is(serveError, http.ErrServerClosed) {
		serveError = nil
	}
	return multierr.Combine(callbackError, serveError, closeError)
}

func (s *serveH1H2) gracefulShutdown(ctx context.Context) {
	// Stop all listeners on the server and poll idle connections to close.
	// If the context expires, Shutdown returns before the server actually
	// stops and the error is non-nil.
	err := s.h.Shutdown(ctx)

	// Shutdown returns the context’s error, otherwise it returns any error
	// returned from closing the Server’s underlying listener on successful
	// shutdown.
	if err == nil || ctx.Err() == nil {
		return
	}

	// If the context expired during Shutdown, we also want to forcefully
	// close all connections.
	_ = s.h.Close()
}

type onceCloseListener struct {
	net.Listener
	once sync.Once
	err  error
}

func (l *onceCloseListener) Close() error {
	l.once.Do(func() { l.err = l.Listener.Close() })
	return l.err
}
