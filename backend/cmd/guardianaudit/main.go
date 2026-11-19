package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"time"

	"school-management-system/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// guardianaudit performs read-only duplicate checks required before V51.
// It deliberately omits full contact values from its output.
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL.String())
	if err != nil {
		return fmt.Errorf("connect to configured database: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping configured database: %w", err)
	}

	fmt.Printf("Database: %s\n", cfg.DBName)
	fmt.Println("Read-only guardian duplicate audit; no data or schema changes will be made.")

	if err := reportDuplicateGroups(ctx, pool, "mobile", `
		WITH normalized AS (
			SELECT school_id, id, full_name,
			       CASE\n\t\t\t\tWHEN length(regexp_replace(mobile_number, '[^0-9]', '', 'g')) = 12\n\t\t\t\t     AND left(regexp_replace(mobile_number, '[^0-9]', '', 'g'), 2) = '91'\n\t\t\t\tTHEN right(regexp_replace(mobile_number, '[^0-9]', '', 'g'), 10)\n\t\t\t\tWHEN length(regexp_replace(mobile_number, '[^0-9]', '', 'g')) = 11\n\t\t\t\t     AND left(regexp_replace(mobile_number, '[^0-9]', '', 'g'), 1) = '0'\n\t\t\t\tTHEN right(regexp_replace(mobile_number, '[^0-9]', '', 'g'), 10)\n\t\t\t\tELSE NULLIF(regexp_replace(mobile_number, '[^0-9]', '', 'g'), '')\n\t\t\tEND AS contact_key
			FROM guardians
			WHERE mobile_number IS NOT NULL
		), duplicates AS (
			SELECT school_id, contact_key, COUNT(*) AS record_count,
			       string_agg(id::text || ' (' || full_name || ')', '; ' ORDER BY id) AS records
			FROM normalized
			WHERE contact_key IS NOT NULL
			GROUP BY school_id, contact_key
			HAVING COUNT(*) > 1
		)
		SELECT school_id, right(contact_key, 4), record_count, records
		FROM duplicates
		ORDER BY school_id, right(contact_key, 4)
	`); err != nil {
		return err
	}

	if err := reportDuplicateGroups(ctx, pool, "email", `
		WITH normalized AS (
			SELECT school_id, id, full_name,
			       lower(btrim(email_address)) AS contact_key
			FROM guardians
			WHERE email_address IS NOT NULL AND btrim(email_address) <> ''
		), duplicates AS (
			SELECT school_id, contact_key, COUNT(*) AS record_count,
			       string_agg(id::text || ' (' || full_name || ')', '; ' ORDER BY id) AS records
			FROM normalized
			GROUP BY school_id, contact_key
			HAVING COUNT(*) > 1
		)
		SELECT school_id,
		       left(contact_key, 1) || '***@' || split_part(contact_key, '@', 2),
		       record_count, records
		FROM duplicates
		ORDER BY school_id, left(contact_key, 1)
	`); err != nil {
		return err
	}

	fmt.Println("Audit complete. V51 has NOT been applied.")
	return nil
}

func reportDuplicateGroups(ctx context.Context, pool *pgxpool.Pool, kind, sql string) error {
	rows, err := pool.Query(ctx, sql)
	if err != nil {
		return fmt.Errorf("audit guardian %s duplicates: %w", kind, err)
	}
	defer rows.Close()

	groups := 0
	fmt.Printf("\nGuardian %s duplicate groups:\n", kind)
	for rows.Next() {
		var schoolID int64
		var maskedKey string
		var count int
		var records string
		if err := rows.Scan(&schoolID, &maskedKey, &count, &records); err != nil {
			return fmt.Errorf("read guardian %s audit row: %w", kind, err)
		}
		groups++
		fmt.Printf("  school_id=%d key=%s records=%d guardians=[%s]\n", schoolID, maskedKey, count, records)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate guardian %s audit rows: %w", kind, err)
	}
	if groups == 0 {
		fmt.Println("  none")
	} else {
		fmt.Printf("  total groups: %d\n", groups)
	}
	return nil
}
