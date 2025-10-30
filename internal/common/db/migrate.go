package db

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"ride-hail/internal/common/logger"
	"sort"
	"strings"
	"time"
)

func (p *Postgres) RunMigrations(migrationsDir string) error {
	start := time.Now()
	logger.Info("db_migrations_start", "Running database migrations...", "", "")

	files, err := readMigrationFiles(migrationsDir)
	if err != nil {
		logger.Error("db_migrations_read_failed", "Failed to read migration files", "", "", err.Error())
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	_, err = p.Conn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS _migrations (
			id SERIAL PRIMARY KEY,
			filename TEXT UNIQUE NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		logger.Error("db_migrations_table_failed", "Failed to ensure _migrations table", "", "", err.Error())
		return fmt.Errorf("failed to create _migrations table: %w", err)
	}

	executed := make(map[string]bool)
	rows, err := p.Conn.Query(context.Background(), "SELECT filename FROM _migrations")
	if err != nil {
		logger.Error("db_migrations_query_failed", "Failed to fetch applied migrations", "", "", err.Error())
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var fname string
		if err := rows.Scan(&fname); err != nil {
			return err
		}
		executed[fname] = true
	}

	for _, file := range files {
		name := filepath.Base(file)
		if executed[name] {
			logger.Info("db_migration_skip", fmt.Sprintf("Skipping already applied migration: %s", name), "", "")
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			logger.Error("db_migration_read_file_failed", fmt.Sprintf("Failed to read migration file %s", name), "", "", err.Error())
			return fmt.Errorf("failed to read migration file %s: %w", name, err)
		}

		logger.Info("db_migration_apply", fmt.Sprintf("Applying migration: %s", name), "", "")
		tx, err := p.Conn.Begin(context.Background())
		if err != nil {
			return fmt.Errorf("failed to start migration transaction: %w", err)
		}

		if _, err := tx.Exec(context.Background(), string(content)); err != nil {
			tx.Rollback(context.Background())
			logger.Error("db_migration_failed", fmt.Sprintf("Migration %s failed", name), "", "", err.Error())
			return fmt.Errorf("migration %s failed: %w", name, err)
		}

		if _, err := tx.Exec(context.Background(),
			"INSERT INTO _migrations (filename) VALUES ($1)", name); err != nil {
			tx.Rollback(context.Background())
			return fmt.Errorf("failed to record migration %s: %w", name, err)
		}

		if err := tx.Commit(context.Background()); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", name, err)
		}
	}

	logger.Info("db_migrations_done", fmt.Sprintf("Migrations applied successfully in %v", time.Since(start)), "", "")
	return nil
}

func readMigrationFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && (strings.HasSuffix(d.Name(), ".up.sql") || strings.HasSuffix(d.Name(), ".up")) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}
