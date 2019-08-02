package testutils

import (
	"database/sql"
	"os"
	"path"
	"testing"

	"github.com/golang-migrate/migrate/v4"

	// Needed for the migrations
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func clearDB(t *testing.T, dbURL string) {
	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		t.Fatalf("Error while connecting to database: %s", err)
	}

	defer db.Close()

	tables := []string{"users", "blogs"}

	tx, err := db.Begin()

	if err != nil {
		t.Fatalf("Error while starting transaction: %s", err)
	}

	for _, table := range tables {
		if _, err := tx.Exec("TRUNCATE TABLE " + table + " CASCADE"); err != nil {
			t.Fatalf("Error while truncating table %s: %s", table, err)
		}
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Error while committing changes: %s", err)
	}
}

func migrateSchema(t *testing.T, dbURL string) {
	wd, err := os.Getwd()

	if err != nil {
		t.Fatalf("Error while retrieving working directory: %s", err)
	}

	migrationsPath := path.Clean(path.Join(wd, "..", "..", "sql"))

	migration, err := migrate.New("file://"+migrationsPath, dbURL)

	if err != nil {
		t.Fatalf("Error while initializing migration: %s", err)
	}

	if err := migration.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Error while setting up schema: %s", err)
	}
}

func FlushDB(t *testing.T) {
	dbURL := os.Getenv(DBURLEnvVar)

	if dbURL == "" {
		return
	}

	migrateSchema(t, dbURL)
	clearDB(t, dbURL)
}
