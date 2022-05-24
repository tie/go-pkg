package httpserver

import (
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
)

// exitOnPanicHandler returns a handler that is a workaround for
// https://go.dev/issues/16542 and https://go.dev/issue/37920.
//
// Go net/http server implementation recovers from panics. For some servers
// this may leave the application in inconsistent state until someone notices
// panic in logs (if they were enabled).
//
// See also https://iximiuz.com/en/posts/go-http-handlers-panic-and-deadlocks/
func exitOnPanicHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer exitOnPanic(http.ErrAbortHandler)
		h.ServeHTTP(w, r)
	}
}

// exitOnPanic prints stack trace and calls os.Exit(2) if it recovers from
// panic. It is intended to be used in deferred calls to avoid propagating
// panics to the callers.
//
// As a special case, it if it recovers from http.ErrAbortHandler error, the
// panic is propagated to the caller.
func exitOnPanic(except ...any) {
	e := recover()
	if e == nil {
		return
	}
	for _, v := range except {
		if e == v {
			return
		}
	}

	// TODO(tie): match the output of Go runtime.
	//
	// In particular, it uses different value formatting and recovering
	// from panic adds a more stack frames (and debug.Stack() does too).
	// It also uses builtin print and println functions for output that
	// cannot be redirected by changing os.Stderr.
	//
	// Since the output is slightly different, we add a greppable panic
	// prefix for now.
	fmt.Fprintf(os.Stderr, "panic in http.Handler: %v\n\n%s", e, debug.Stack())
	os.Exit(2)
}
