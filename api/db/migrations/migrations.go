package migrations

import (
	"database/sql"
	"embed"
	"mrtutor/api/db"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var fs embed.FS

var MigrateClient migrate.Migrate

func NewWithDb(db *sql.DB) (*migrate.Migrate, error) {
	source, err := iofs.New(fs, ".")
	if err != nil {
		return nil, err
	}
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, err
	}
	m, err := migrate.NewWithInstance("iofs", source, "sqlite3", driver)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func NewMigrate() (*migrate.Migrate, error) {
	db, err := db.New()
	if err != nil {
		return nil, err
	}
	return NewWithDb(db)
}
