// Command server is book-content's content-serving REST API: it reads
// book/chunk/question/breakdown data that the pipeline (a separate Go
// module, see /pipeline) already wrote to Postgres, and serves it over
// HTTP for the Flutter app (/app). It never writes anything — no
// endpoint here calls the Claude API or mutates a row; see
// book-content/internal/api's routes.
//
// Requires the local dev Postgres to be up (`docker compose up -d` from
// the repo root). Reads POSTGRES_* from the real environment or a .env
// file at the repo root (see shared/dotenv) — same variables and
// defaults pipeline/cmd/process uses, so both point at the same
// database with no extra configuration.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aidoku/book-content/internal/api"
	"aidoku/book-content/internal/db"
	"aidoku/shared/dotenv"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	flag.Parse()

	if err := run(*addr); err != nil {
		fmt.Fprintln(os.Stderr, "server:", err)
		os.Exit(1)
	}
}

func run(addr string) error {
	dotenv.Load(".env")
	dotenv.Load("../.env") // in case run from book-content/

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, db.ConnStringFromEnv())
	if err != nil {
		return fmt.Errorf("connect to Postgres (is `docker compose up -d` running?): %w", err)
	}
	defer pool.Close()

	store := db.New(pool)
	mux := api.NewRouter(store)

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("server: listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		log.Print("server: shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	}
}
