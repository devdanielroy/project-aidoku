// Package db persists pipeline output — books, chunks, questions, and
// (once that stage exists) breakdowns — to Postgres. See db/schema.sql
// at the repo root for the actual DDL and AIDOKU_DESIGN.md §4/§6 for the
// data model and storage decision.
//
// Nothing in this package talks to the Anthropic API; it's purely the
// write side for whatever earlier pipeline stages already produced.
package db

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// conn is the subset of *pgxpool.Pool that Store needs. pgx.Tx satisfies
// it too (its Begin opens a savepoint-based nested transaction) — so a
// Store can wrap either a real pool (normal use) or an open test
// transaction (tests, rolled back afterward instead of leaving rows
// behind), with no behavior change to the SQL itself. Same
// consumer-defined-interface-for-testability pattern as llmCaller in
// internal/chunk and internal/question.
type conn interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Store persists pipeline output to Postgres.
type Store struct {
	db conn
}

// New wraps db (a *pgxpool.Pool in normal use, or a pgx.Tx in tests) in a
// Store.
func New(db conn) *Store {
	return &Store{db: db}
}

// Open connects to Postgres at connString (see ConnStringFromEnv) and
// verifies the connection with a ping before returning. Callers own the
// returned pool's lifetime — Close it when done.
func Open(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("db: open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return pool, nil
}

// ConnStringFromEnv builds a Postgres connection string from POSTGRES_*
// environment variables, defaulting to the same values
// docker-compose.yml's local dev Postgres falls back to when they're
// unset — so this works out of the box against `docker compose up -d`
// with no configuration. POSTGRES_HOST defaults to "localhost" (not one
// of docker-compose.yml's own vars — compose addresses the container by
// service name internally; this is for callers running outside Docker,
// connecting to the port compose publishes on the host).
func ConnStringFromEnv() string {
	host := envOr("POSTGRES_HOST", "localhost")
	port := envOr("POSTGRES_PORT", "5432")
	user := envOr("POSTGRES_USER", "aidoku")
	password := envOr("POSTGRES_PASSWORD", "aidoku_dev")
	name := envOr("POSTGRES_DB", "aidoku")

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		url.QueryEscape(user), url.QueryEscape(password), host, port, name)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// compile-time check that *pgxpool.Pool satisfies conn.
var _ conn = (*pgxpool.Pool)(nil)
