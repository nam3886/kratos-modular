// Package testkit gives tests the real services the application runs on.
// Each helper starts a container once per test binary. To use services that
// are already running instead, set TEST_DATABASE_URL and TEST_RABBITMQ_URL;
// packages then share them, so run the tests with -p 1.
package testkit

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/nam3886/kratos-modular/internal/conf"
	"github.com/nam3886/kratos-modular/internal/platform/db"
	"github.com/nam3886/kratos-modular/internal/platform/migrate"
)

var (
	pgOnce sync.Once
	pgURL  string
	pgErr  error
)

func PostgresURL(t testing.TB) string {
	t.Helper()
	pgOnce.Do(func() {
		if pgURL = os.Getenv("TEST_DATABASE_URL"); pgURL != "" {
			// Tests drop the whole public schema.
			if u, err := url.Parse(pgURL); err != nil || !strings.Contains(u.Path, "test") {
				pgErr = fmt.Errorf("TEST_DATABASE_URL must name a database containing %q, because tests wipe it", "test")
			}
			return
		}
		ctx := context.Background()
		ctr, err := postgres.Run(ctx, "postgres:17-alpine",
			postgres.WithDatabase("test"),
			postgres.WithUsername("test"),
			postgres.WithPassword("test"),
			postgres.BasicWaitStrategies(),
		)
		if err != nil {
			pgErr = err
			return
		}
		// The container is removed by testcontainers' reaper when the test binary exits.
		pgURL, pgErr = ctr.ConnectionString(ctx, "sslmode=disable")
	})
	if pgErr != nil {
		t.Fatalf("start postgres: %v", pgErr)
	}
	return pgURL
}

// Postgres returns a manager whose "main" database is empty and migrated.
func Postgres(t testing.TB) *db.Manager {
	t.Helper()
	dbs, cleanup, err := db.Open(&conf.Data{Databases: map[string]*conf.Data_Database{
		"main": {Driver: "pgx", Source: PostgresURL(t)},
	}})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(cleanup)
	ResetPostgres(t, dbs)
	return dbs
}

// ResetPostgres drops everything in the "main" database and migrates it again.
func ResetPostgres(t testing.TB, dbs *db.Manager) {
	t.Helper()
	main, err := dbs.Get("main")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := main.Pool.ExecContext(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatalf("reset test database: %v", err)
	}
	if err := migrate.Up(ctx, dbs, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
}

func Count(t testing.TB, dbs *db.Manager, table string) int {
	t.Helper()
	main, err := dbs.Get("main")
	if err != nil {
		t.Fatal(err)
	}
	var n int
	if err := main.Pool.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
