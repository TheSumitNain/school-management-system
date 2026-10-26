package main

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"

	"school-management-system/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {

	cfg := config.Load()

	if err := cfg.ValidateDatabase(); err != nil {
		log.Fatal(err)
	}

	dbURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Host:   net.JoinHostPort(cfg.DBHost, cfg.DBPort),
		Path:   "/" + cfg.DBName,
	}

	query := dbURL.Query()
	query.Set("sslmode", cfg.DBSSLMode)
	dbURL.RawQuery = query.Encode()

	migrationsPath, err := filepath.Abs("migrations")
	if err != nil {
		log.Fatal("Could not locate migrations directory")
	}

	migrationsPath = filepath.ToSlash(migrationsPath)

	sourceURL := "file:" + migrationsPath

	m, err := migrate.New(sourceURL, dbURL.String())
	if err != nil {
		log.Fatal("Could not initialize migrations: ", err)
	}
	defer func() {
		sourceErr, databaseErr := m.Close()
		if sourceErr != nil {
			log.Println("Migration source close error:", sourceErr)
		}
		if databaseErr != nil {
			log.Println("Migration database close error:", databaseErr)
		}
	}()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("Migration failed: ", err)
	}

	fmt.Fprintln(os.Stdout, "Database migrations completed successfully")
}
