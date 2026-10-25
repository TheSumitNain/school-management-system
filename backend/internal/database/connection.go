package database

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"school-management-system/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	if cfg.DBHost == "" ||
		cfg.DBPort == "" ||
		cfg.DBUser == "" ||
		cfg.DBName == "" ||
		cfg.DBPassword == "" {
		return nil, fmt.Errorf("database configuration is incomplete")
	}

	port := cfg.DBPort
	if _, err := net.LookupPort("tcp", port); err != nil {
		return nil, fmt.Errorf("invalid database port: %w", err)
	}

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Host:   net.JoinHostPort(cfg.DBHost, port),
		Path:   "/" + cfg.DBName,
	}

	query := u.Query()
	query.Set("sslmode", "disable")
	u.RawQuery = query.Encode()

	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	return pool, nil
}
