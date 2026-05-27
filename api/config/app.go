package config

import (
	"cmp"
	"os"
	"strings"
)

type AppMode string

const (
	DEV  AppMode = "dev"
	PROD AppMode = "prod"
	TEST AppMode = "test"
)

var (
	Mode AppMode = cmp.Or(parseAppMode(os.Getenv("APP_ENV")), DEV)
)

func parseAppMode(mode string) AppMode {
	mode = strings.ToLower(mode)
	switch mode {
	case "dev":
	case "development":
		return DEV
	case "prod":
	case "production":
		return PROD
	case "test":
	case "testing":
		return TEST
	default:
		return ""
	}
	return ""
}
