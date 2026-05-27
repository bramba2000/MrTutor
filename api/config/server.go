package config

import (
	"cmp"
	"os"
	"strconv"
	"time"
)

var (
	Port            = cmp.Or(os.Getenv("PORT"), "8080")
	ApiBasePath     = cmp.Or(os.Getenv("BASEPATH"), "/api/v0")
	ShutdownTimeout = cmp.Or(getDurationEnv("SHUTDOWN_TIMEOUT"), 5*time.Second)
)

func getIntEnv(key string) int {
	value := os.Getenv(key)
	strconv, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return strconv
}

func getDurationEnv(key string) time.Duration {
	value := os.Getenv(key)
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0
	}
	return duration
}
