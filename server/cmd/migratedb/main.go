package main

import (
	"flag"
	"log"
	"os"

	"github.com/mattes/migrate"
	_ "github.com/mattes/migrate/database/postgres"
	"github.com/mattes/migrate/source/go-bindata"

	"github.com/abustany/moblog-cloud/sql"
)

var dbURL = flag.String("db", "", "URL to the PostgreSQL server. If not set, the DB_URL environment variable is used.")

func main() {
	flag.Parse()

	if *dbURL == "" {
		*dbURL = os.Getenv("DB_URL")
	}

	if *dbURL == "" {
		log.Fatal("No database URL set. Use -db or the DB_URL environment variable.")
	}

	s := bindata.Resource(sql.AssetNames(),
		func(name string) ([]byte, error) {
			return sql.Asset(name)
		})

	d, err := bindata.WithInstance(s)

	if err != nil {
		log.Fatalf("Error while initializing bindata: %s", err)
	}

	m, err := migrate.NewWithSourceInstance("go-bindata", d, *dbURL)

	if err != nil {
		log.Fatalf("Error while initializing migration: %s", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Error while running migration: %s", err)
	}

	log.Printf("Migration done")
}
