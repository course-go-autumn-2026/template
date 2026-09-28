package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	appconfig "github.com/imrtlDrifter/template/internal/config"
	"github.com/imrtlDrifter/template/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg, setErr := appconfig.Load()
	if setErr != nil {
		log.Printf("load config: %v", setErr)
		os.Exit(1)
	}

	if err := apply(cfg); err != nil {
		log.Printf("migration failed: %v", err)
		os.Exit(1)
	}

	log.Println("migration applied")
}

func apply(cfg *appconfig.Configuration) error {
	db, err := sql.Open("pgx", cfg.PostgreSQL.URL)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer func(db *sql.DB) {
		closeErr := db.Close()
		if closeErr != nil {
			log.Printf("close db: %v", closeErr)
		}
	}(db)

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("iofs source: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("postgres driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "up":
		err = m.Up()
	case "down":
		err = m.Steps(-1)
	default:
		return fmt.Errorf("unknown migration command %q", command)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate %s: %w", command, err)
	}

	return nil
}
