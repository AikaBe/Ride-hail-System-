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
		logger.Error("db_migrations_read_failed", "Failed to read migration files", "", "", err.Error(), "")
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	tx, err := p.Conn.Begin(context.Background())
	if err != nil {
		logger.Error("db_migrations_tx_failed", "Failed to start migration transaction", "", "", err.Error(), "")
		return fmt.Errorf("failed to start migration transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			logger.Error("db_migration_read_file_failed", fmt.Sprintf("Failed to read migration file %s", file), "", "", err.Error(), "")
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		logger.Info("db_migration_apply", fmt.Sprintf("Applying migration: %s", filepath.Base(file)), "", "")
		if _, err := tx.Exec(context.Background(), string(content)); err != nil {
			logger.Error("db_migration_failed", fmt.Sprintf("Migration %s failed", file), "", "", err.Error(), "")
			return fmt.Errorf("migration %s failed: %w", file, err)
		}
	}

	if err := tx.Commit(context.Background()); err != nil {
		logger.Error("db_migrations_commit_failed", "Failed to commit migrations", "", "", err.Error(), "")
		return fmt.Errorf("failed to commit migrations: %w", err)
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
