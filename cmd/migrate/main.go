// Command migrate brings every database in the config up to date.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/go-kratos/kratos/v3/config"
	"github.com/go-kratos/kratos/v3/config/file"

	"github.com/nam3886/kratos-modular/internal/conf"
	"github.com/nam3886/kratos-modular/internal/platform/db"
	"github.com/nam3886/kratos-modular/internal/platform/migrate"
)

func main() {
	flagconf := flag.String("conf", "../../configs", "config path, eg: -conf config.yaml")
	flag.Parse()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(*flagconf, log); err != nil {
		log.Error("migrate", "error", err)
		os.Exit(1)
	}
}

func run(path string, log *slog.Logger) error {
	c := config.New(config.WithSource(file.NewSource(path)))
	defer func() { _ = c.Close() }()
	if err := c.Load(); err != nil {
		return err
	}
	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		return err
	}
	dbs, cleanup, err := db.Open(bc.Data)
	if err != nil {
		return err
	}
	defer cleanup()
	return migrate.Up(context.Background(), dbs, log)
}
