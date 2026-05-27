package main

import (
	"mrtutor-api/config"
	"net/http"
	"strings"
)

func healthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isShuttingDownServer.Load() {
			http.Error(w, "Server is shutting down", http.StatusServiceUnavailable)
		}
		w.WriteHeader(http.StatusOK)
	})
}

func addRoutes(mux *http.ServeMux) {
	internalMux := http.NewServeMux()
	internalMux.Handle("/health", healthHandler())

	basePath := strings.TrimRight(config.ApiBasePath, "/")
	if basePath == "" {
		basePath = "/"
	}

	if basePath == "/" {
		mux.Handle("/", internalMux)
		return
	}

	mux.Handle(basePath+"/", http.StripPrefix(basePath, internalMux))
}
