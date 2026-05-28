package main

import (
	"log/slog"
	"net/http"
)

// applyMiddleware takes a handler and a list of middleware functions, and applies the middleware to
// the handler in reverse order (flow follow provided list).
func applyMiddleware(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

type Middleware func(http.Handler) http.Handler

func newLoggingMiddleware(logger *slog.Logger) Middleware {
	logger = logger.With("component", "loggingMiddleware")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info("Request received", "method", r.Method, "url", r.URL.String())
			next.ServeHTTP(w, r)
		})
	}
}
