package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/spf13/cobra"
)

const (
	dialect = "postgres"
)

var (
	dryRun           bool
	migrateTableName string
	migrateDir       string
	migrateSchema    string
	max              int
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "schema apply command",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("\n=== Running migration for main database: %s ===\n", Opt.DB)
		if err := runMigration(Opt.DB); err != nil {
			return fmt.Errorf("failed to migrate main database: %w", err)
		}

		return nil
	},
}

func runMigration(dbName string) error {
	smode := "disable"
	if Opt.SSLMode {
		smode = "verify-ca"
	}
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		Opt.Host, Opt.User, Opt.Password, dbName, Opt.Port, smode, Opt.TimeZone,
	)
	db, err := sql.Open(dialect, dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	migrate.SetSchema(migrateSchema)
	migrate.SetTable(migrateTableName)
	src := migrate.FileMigrationSource{
		Dir: migrateDir,
	}

	if dryRun {
		migrations, _, err := migrate.PlanMigration(db, dialect, src, migrate.Up, max)
		if err != nil {
			return fmt.Errorf("Cannot plan migration: %s", err)
		}
		for _, m := range migrations {
			PrintMigration(m, migrate.Up)
		}
		return nil
	}

	n, err := migrate.ExecMax(db, dialect, src, migrate.Up, max)
	if err != nil {
		return err
	}
	fmt.Printf("Applied %d migrations to database '%s'\n", n, dbName)
	return nil
}

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Insert seed data into the database",
	RunE: func(cmd *cobra.Command, args []string) error {
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=%s",
			Opt.Host, Opt.User, Opt.Password, Opt.DB, Opt.Port, Opt.TimeZone,
		)
		db, err := sql.Open(dialect, dsn)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}
		execDir := filepath.Dir(execPath)
		if strings.Contains(execDir, "go-build") || strings.Contains(execDir, "/tmp/") {
			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}
			execDir = workDir
		}

		seedFile := filepath.Join(execDir, "database", "migration", "seed", "seed_data.sql")
		content, err := os.ReadFile(filepath.Clean(seedFile))
		if err != nil {
			return fmt.Errorf("failed to read seed file (%s): %w", seedFile, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute seed queries: %w", err)
		}

		fmt.Println("Seed data applied successfully!")
		return nil
	},
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "schema down command",
	RunE: func(cmd *cobra.Command, args []string) error {
		smode := "disable"
		if Opt.SSLMode {
			smode = "verify-ca"
		}
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
			Opt.Host, Opt.User, Opt.Password, Opt.DB, Opt.Port, smode, Opt.TimeZone,
		)
		db, err := sql.Open(dialect, dsn)
		if err != nil {
			return err
		}
		defer db.Close()

		migrate.SetSchema(migrateSchema)
		migrate.SetTable(migrateTableName)
		src := migrate.FileMigrationSource{
			Dir: migrateDir,
		}

		if dryRun {
			migrations, _, err := migrate.PlanMigration(db, dialect, src, migrate.Down, max)
			if err != nil {
				return fmt.Errorf("Cannot plan migration: %s", err)
			}
			for _, m := range migrations {
				PrintMigration(m, migrate.Down)
			}
			return nil
		}

		n, err := migrate.ExecMax(db, dialect, src, migrate.Down, max)
		if err != nil {
			return err
		}
		fmt.Printf("Applied %d migrations", n)
		return nil
	},
}

func init() {
	applyCmd.PersistentFlags().StringVarP(&migrateTableName, "table", "t", "migrations", "migration table name")
	applyCmd.PersistentFlags().StringVarP(&migrateDir, "dir", "r", "", "migration directory")
	applyCmd.PersistentFlags().StringVar(&migrateSchema, "schema-name", "uasl_reservation", "schema name for migration tracking table")
	applyCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "d", false, "dry run mode")
	applyCmd.PersistentFlags().IntVarP(&max, "max", "m", 0, "limit of apply migration")
	SchemaCmd.AddCommand(applyCmd)

	downCmd.PersistentFlags().StringVarP(&migrateTableName, "table", "t", "migrations", "migration table name")
	downCmd.PersistentFlags().StringVarP(&migrateDir, "dir", "r", "", "migration directory")
	downCmd.PersistentFlags().StringVar(&migrateSchema, "schema-name", "uasl_reservation", "schema name for migration tracking table")
	downCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "d", false, "dry run mode")
	downCmd.PersistentFlags().IntVarP(&max, "max", "m", 0, "limit of down migration")
	SchemaCmd.AddCommand(downCmd)

	RootCmd.AddCommand(seedCmd)
}

func PrintMigration(m *migrate.PlannedMigration, dir migrate.MigrationDirection) {
	fmt.Printf("==> Would apply migration %s (up)", m.Id)
	for _, q := range m.Up {
		fmt.Println(q)
	}
}
