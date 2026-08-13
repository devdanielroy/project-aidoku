// Package db is the read-only query layer this server uses to serve
// book/chunk/question/breakdown content out of Postgres to the Flutter
// app. See db/schema.sql at the repo root for the actual DDL.
//
// This is a separate Go module from pipeline/internal/db (its write-side
// counterpart) — see AIDOKU_DESIGN.md's backend handoff notes for why
// the two aren't shared, though the connection-level plumbing (conn
// interface, Open, ConnStringFromEnv) is, via shared/dbconn. Every query
// here also enforces that a book is published (see store.go) — nothing
// pipeline-generated but not yet QA'd/published is ever reachable
// through this API.
package db

import (
	"context"

	"aidoku/shared/dbconn"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// conn is dbconn.Conn under a shorter local name.
type conn = dbconn.Conn

// Store reads pipeline output back out of Postgres.
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
	return dbconn.Open(ctx, connString)
}

// ConnStringFromEnv builds a Postgres connection string from POSTGRES_*
// environment variables — see dbconn.ConnStringFromEnv for the defaults.
func ConnStringFromEnv() string {
	return dbconn.ConnStringFromEnv()
}

// compile-time check that *pgxpool.Pool satisfies conn.
var _ conn = (*pgxpool.Pool)(nil)

// compile-time check that pgx.Tx satisfies conn.
var _ conn = (pgx.Tx)(nil)
