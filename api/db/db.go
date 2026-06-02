package db

import (
	"database/sql"
	"mrtutor/api/config"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const (
	driverName = "sqlite3"
)

// New returns a new database connection using the DSN from [config.DSN]
func New() (*sql.DB, error) {
	return sql.Open("sqlite3", config.DSN)
}

// NewInMemory returns a new in-memory database connection using [config.DSN] options if they exist.
func NewInMemory() (*sql.DB, error) {
	_, options, found := strings.Cut(config.DSN, "?")
	if found {
		return sql.Open(driverName, ":memory:"+"?"+options)
	}
	return sql.Open(driverName, ":memory:")
}
