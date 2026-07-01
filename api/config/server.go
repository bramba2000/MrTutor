package config

import (
	"cmp"
	"os"
	"time"
)

var (
	// Port is the port on which the server will listen for incoming requests.
	Port = cmp.Or(os.Getenv("PORT"), "8080")
	// ApiBasePath is the base path for all API endpoints.
	ApiBasePath = cmp.Or(os.Getenv("BASEPATH"), "/api/v0")
	// ShutdownTimeout is the duration to wait for the server to shut down gracefully before forcing it to close.
	ShutdownTimeout = cmp.Or(getDurationEnv("SHUTDOWN_TIMEOUT"), 5*time.Second)
)

func getDurationEnv(key string) time.Duration {
	value := os.Getenv(key)
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0
	}
	return duration
}
