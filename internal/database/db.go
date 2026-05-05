package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"alphapulse/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func New(ctx context.Context, cfg *config.Config, migrationPath string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if err := RunMigrations(ctx, pool, migrationPath); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationPath string) error {
	info, err := os.Stat(migrationPath)
	if err != nil {
		return fmt.Errorf("stat migration path: %w", err)
	}

	paths := []string{migrationPath}
	if info.IsDir() {
		entries, err := os.ReadDir(migrationPath)
		if err != nil {
			return fmt.Errorf("read migration dir: %w", err)
		}

		paths = make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
				continue
			}
			paths = append(paths, filepath.Join(migrationPath, entry.Name()))
		}
		sort.Strings(paths)
	}

	for _, path := range paths {
		migrationSQL, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration file %s: %w", path, err)
		}
		if _, err := pool.Exec(ctx, string(migrationSQL)); err != nil {
			return fmt.Errorf("run migration %s: %w", path, err)
		}
	}

	return nil
}

func EnsureAdminUser(ctx context.Context, pool *pgxpool.Pool, username, password string) (bool, error) {
	var existingID string
	err := pool.QueryRow(ctx, `SELECT id FROM users WHERE username = $1`, username).Scan(&existingID)
	switch {
	case err == nil:
		return false, nil
	case err != pgx.ErrNoRows:
		return false, fmt.Errorf("check admin user: %w", err)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return false, fmt.Errorf("hash admin password: %w", err)
	}

	_, err = pool.Exec(
		ctx,
		`INSERT INTO users (username, password_hash, role) VALUES ($1, $2, 'admin')`,
		username,
		string(passwordHash),
	)
	if err == nil {
		return true, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return false, nil
	}

	return false, fmt.Errorf("create admin user: %w", err)
}
