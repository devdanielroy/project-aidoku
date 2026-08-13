// Package dbconn is the shared Postgres connection plumbing behind both
// pipeline/internal/db (write side) and book-content/internal/db (read side) —
// two separate Go modules, wired together via go.work at the repo root.
// Each module still defines its own Store type and query methods (they
// genuinely differ: one upserts pipeline output, the other serves it
// back out as JSON) — only the connection-level pieces that were
// actually byte-for-byte identical live here.
package dbconn

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Conn is the subset of *pgxpool.Pool a Store needs. pgx.Tx satisfies it
// too (its Begin opens a savepoint-based nested transaction), so a Store
// can wrap either a real pool (normal use) or an open test transaction
// (tests, rolled back afterward) with no behavior change to the SQL
// itself. A superset of what either module's Store actually calls
// (write side never Query, read side never Exec) — fine, since both
// *pgxpool.Pool and pgx.Tx already implement all four methods; a wider
// shared interface costs nothing here.
type Conn interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Open connects to Postgres at connString (see ConnStringFromEnv) and
// verifies the connection with a ping before returning. Callers own the
// returned pool's lifetime — Close it when done.
func Open(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("dbconn: open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("dbconn: ping: %w", err)
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

// compile-time check that *pgxpool.Pool satisfies Conn.
var _ Conn = (*pgxpool.Pool)(nil)

// compile-time check that pgx.Tx satisfies Conn.
var _ Conn = (pgx.Tx)(nil)
