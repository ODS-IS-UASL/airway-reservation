package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"
	"gorm.io/gorm"
)

const (
	migrationDialect = "postgres"
	migrationSchema  = "uasl_reservation"
	migrationTable   = "migrations"
	migrationDir     = "database/migration/uasl_reservation"
	seedFile         = "database/migration/seed/seed_data.sql"
)

// RunMigrations runs all pending migrations against the database connected via gormDB.
// If enableSeed is true (e.g. ENABLE_SEED=true), seed data is also applied.
func RunMigrations(gormDB *gorm.DB, enableSeed bool) error {
	sqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from gorm: %w", err)
	}

	baseDir := resolveBaseDir()

	if err := runMigrate(sqlDB, baseDir); err != nil {
		return err
	}

	if enableSeed {
		if err := runSeed(sqlDB, baseDir); err != nil {
			return err
		}
	}

	return nil
}

func resolveBaseDir() string {
	execPath, err := os.Executable()
	if err != nil {
		return "."
	}
	execDir := filepath.Dir(execPath)
	if strings.Contains(execDir, "go-build") || strings.Contains(execDir, "tmp") {
		if wd, err := os.Getwd(); err == nil {
			return wd
		}
	}
	return execDir
}

func runMigrate(db *sql.DB, baseDir string) error {
	dir := filepath.Join(baseDir, migrationDir)

	// geometry 型など postgis の型を uasl_reservation スキーマから参照できるよう search_path を設定
	if _, err := db.Exec("SET search_path TO uasl_reservation, public"); err != nil {
		return fmt.Errorf("failed to set search_path: %w", err)
	}

	migrate.SetSchema(migrationSchema)
	migrate.SetTable(migrationTable)
	src := migrate.FileMigrationSource{
		Dir: dir,
	}

	n, err := migrate.ExecMax(db, migrationDialect, src, migrate.Up, 0)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	fmt.Printf("Applied %d migration(s)\n", n)
	return nil
}

func runSeed(db *sql.DB, baseDir string) error {
	path := filepath.Join(baseDir, seedFile)
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("failed to read seed file (%s): %w", path, err)
	}

	if _, err := db.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute seed queries: %w", err)
	}

	fmt.Println("Seed data applied successfully!")
	return nil
}
