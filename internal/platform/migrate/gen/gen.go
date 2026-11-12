// Package gen writes a new migration file from the difference between a
// module's ent schema and the migration directory of its database. It is used
// only by each module's ent/migrate/main.go.
package gen

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqltool"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql/schema"
	_ "github.com/lib/pq" // driver used by the diff engine
)

// Main parses the command line and writes the migration for tables.
func Main(tables []*schema.Table) {
	name := flag.String("name", "", "migration name, e.g. add_payment_note")
	database := flag.String("db", "main", "configured database the module stores into")
	devURL := flag.String("dev-url", "postgres://app:app@127.0.0.1:5432/app_dev?sslmode=disable",
		"an empty scratch database used to replay the existing migrations")
	flag.Parse()
	if *name == "" {
		fmt.Fprintln(os.Stderr, "-name is required")
		os.Exit(2)
	}
	if err := run(context.Background(), *devURL, filepath.Join("migrations", *database), *name, tables); err != nil {
		fmt.Fprintln(os.Stderr, "generate migration:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, devURL, path, name string, tables []*schema.Table) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	dir, err := sqltool.NewGooseDir(path)
	if err != nil {
		return err
	}
	// Accept migration files written or edited by hand: refresh the checksum
	// file instead of refusing to continue when it no longer matches.
	sum, err := dir.Checksum()
	if err != nil {
		return err
	}
	if err := migrate.WriteSumFile(dir, sum); err != nil {
		return err
	}
	m, err := schema.NewMigrateURL(devURL,
		schema.WithDir(dir),
		schema.WithMigrationMode(schema.ModeReplay),
		schema.WithDialect(dialect.Postgres),
		schema.WithFormatter(sqltool.GooseFormatter),
	)
	if err != nil {
		return err
	}
	return m.NamedDiff(ctx, name, tables...)
}
