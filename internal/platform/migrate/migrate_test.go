package migrate_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/nam3886/kratos-modular/internal/platform/migrate"
	"github.com/nam3886/kratos-modular/internal/platform/testkit"
)

func TestUpIsRepeatable(t *testing.T) {
	dbs := testkit.Postgres(t) // already migrated once
	if err := migrate.Up(context.Background(), dbs, slog.New(slog.DiscardHandler)); err != nil {
		t.Fatalf("second Up: %v", err)
	}
	if n := testkit.Count(t, dbs, "settings"); n != 0 {
		t.Fatalf("settings rows = %d, want 0", n)
	}
}
