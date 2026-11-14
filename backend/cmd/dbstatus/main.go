package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"time"

	"school-management-system/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// dbstatus is intentionally read-only. It reports the golang-migrate state
// without applying, rolling back, or repairing any migration.
func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	if err := cfg.ValidateDatabase(); err != nil {
		return err
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL.String())
	if err != nil {
		return fmt.Errorf("connect to configured database: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping configured database: %w", err)
	}

	var migrationTableExists bool
	if err := pool.QueryRow(ctx,
		"SELECT to_regclass('schema_migrations') IS NOT NULL",
	).Scan(&migrationTableExists); err != nil {
		return fmt.Errorf("inspect migration metadata: %w", err)
	}

	fmt.Printf("Database: %s\n", cfg.DBName)
	if !migrationTableExists {
		fmt.Println("Migration version: 0 (no schema_migrations table)")
		fmt.Println("Dirty: false")
		fmt.Println("No migration has been recorded for this database.")
		return nil
	}

	var version int64
	var dirty bool
	if err := pool.QueryRow(ctx,
		"SELECT version, dirty FROM schema_migrations LIMIT 1",
	).Scan(&version, &dirty); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("schema_migrations exists but contains no migration state")
		}
		return fmt.Errorf("read migration state: %w", err)
	}

	fmt.Printf("Migration version: %d\n", version)
	fmt.Printf("Dirty: %t\n", dirty)
	if dirty {
		return errors.New("database migration state is dirty; do not run migrations until the cause is investigated")
	}
	fmt.Println("Read-only migration status check completed; no schema changes were made.")
	return nil
}
