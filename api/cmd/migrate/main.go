package main

import (
	"fmt"
	"mrtutor/api/db/migrations"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
)

func main() {
	m, err := migrations.NewMigrate()
	if err != nil {
		panic(fmt.Sprintf("failed to create migrate instance: %v", err))
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		panic(fmt.Sprintf("failed to run migrations: %v", err))
	}

	fmt.Println("Migrations applied successfully")
}
