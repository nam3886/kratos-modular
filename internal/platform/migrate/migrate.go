// Package migrate brings every configured database up to date: the messaging
// tables first, then the versioned SQL migrations of that database.
package migrate

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/pressly/goose/v3"

	"github.com/nam3886/kratos-modular/internal/platform/db"
	"github.com/nam3886/kratos-modular/internal/platform/mq"
	"github.com/nam3886/kratos-modular/migrations"
)

func Up(ctx context.Context, dbs *db.Manager, log *slog.Logger) error {
	for _, d := range dbs.All() {
		if err := mq.InitTables(ctx, d.Pool); err != nil {
			return fmt.Errorf("database %q: %w", d.Name, err)
		}
		dir, err := fs.Sub(migrations.FS, d.Name)
		if err != nil {
			return err
		}
		p, err := goose.NewProvider(goose.DialectPostgres, d.Pool, dir, goose.WithAllowOutofOrder(true))
		if errors.Is(err, goose.ErrNoMigrations) {
			log.Info("no migrations", "database", d.Name)
			continue
		}
		if err != nil {
			return fmt.Errorf("database %q: %w", d.Name, err)
		}
		results, err := p.Up(ctx)
		if err != nil {
			return fmt.Errorf("database %q: %w", d.Name, err)
		}
		for _, r := range results {
			log.Info("applied", "database", d.Name, "migration", r.Source.Path, "duration", r.Duration)
		}
		log.Info("up to date", "database", d.Name, "applied", len(results))
	}
	return nil
}
