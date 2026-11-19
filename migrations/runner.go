package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
)

func Run(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	files, err := fs.Glob(Files, "*.sql")
	if err != nil {
		return fmt.Errorf("find migration files: %w", err)
	}

	sort.Strings(files)

	for _, filename := range files {
		var applied bool

		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM schema_migrations
				WHERE version = $1
			)
		`, filename).Scan(&applied)

		if err != nil {
			return fmt.Errorf("check migration %s: %w", filename, err)
		}

		if applied {
			continue
		}

		content, err := Files.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", filename, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", filename, err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", filename, err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO schema_migrations (version)
			VALUES ($1)
		`, filename)

		if err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", filename, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", filename, err)
		}
	}

	return nil
}
