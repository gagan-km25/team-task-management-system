package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.up.sql
var migrationFS embed.FS

type DB struct{ Pool *pgxpool.Pool }

func Open(ctx context.Context, url string) (*DB, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &DB{Pool: pool}, nil
}
func (d *DB) Close() { d.Pool.Close() }

// Migrate applies every pending migrations/*.up.sql file in filename order,
// recording applied files in schema_migrations. Each migration runs inside a
// transaction so a failed file leaves no partial schema behind. The pool's
// Exec uses the extended protocol, which rejects multi-statement SQL, so each
// file is executed through the underlying connection's simple protocol.
func Migrate(ctx context.Context, d *DB) error {
	if _, err := d.Pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	files, err := fs.Glob(migrationFS, "migrations/*.up.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, name := range files {
		var applied bool
		if err = d.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, name).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := migrationFS.ReadFile(name)
		if err != nil {
			return err
		}
		tx, err := d.Pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err = tx.Conn().Exec(ctx, string(body)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, name); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if err = tx.Commit(ctx); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
	}
	return nil
}

func IsNoRows(err error) bool        { return errors.Is(err, pgx.ErrNoRows) }
func NormalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
