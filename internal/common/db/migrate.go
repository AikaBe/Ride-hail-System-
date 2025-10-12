package db

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (p *Postgres) RunMigrations(migrationsDir string) error {
	start := time.Now()
	log.Println("🚀 Running database migrations...")

	files, err := readMigrationFiles(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	tx, err := p.Pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("failed to start migration transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		log.Printf("📦 Applying migration: %s", filepath.Base(file))
		if _, err := tx.Exec(context.Background(), string(content)); err != nil {
			return fmt.Errorf("migration %s failed: %w", file, err)
		}
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("failed to commit migrations: %w", err)
	}

	log.Printf("✅ Migrations applied successfully in %v", time.Since(start))
	return nil
}

func readMigrationFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".up") {
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
