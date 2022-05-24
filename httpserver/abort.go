package httpserver

import (
	"net/http"
	"sync"
)

// abortableHandler implements an http.Handler that can be stopped. Handling
// request on a stopped handler panics with http.ErrAbortHandler to avoid
// sending an empty response.
type abortableHandler struct {
	h http.Handler

	mu      sync.Mutex
	wg      sync.WaitGroup
	stopped bool
}

// newAbortableHandler returns a new abortableHandler instance.
func newAbortableHandler(h http.Handler) *abortableHandler {
	return &abortableHandler{
		h: h,
	}
}

// ServeHTTP implements the http.Handler interface.
func (h *abortableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	stopped := h.stopped
	if !stopped {
		h.wg.Add(1)
	}
	h.mu.Unlock()
	if stopped {
		panic(http.ErrAbortHandler)
	}

	defer h.wg.Done()
	h.h.ServeHTTP(w, r)
}

// Stop stops the handler and waits for all ongoing ServeHTTP calls to return.
func (h *abortableHandler) Stop() {
	h.mu.Lock()
	h.stopped = true
	h.mu.Unlock()
	h.wg.Wait()
}
