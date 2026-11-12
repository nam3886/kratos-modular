//go:build ignore

// Command main writes a migration for this module's schema changes:
//
//	make migration module=settings name=<what_changed>
package main

import (
	"github.com/nam3886/kratos-modular/internal/module/settings/internal/data/ent/migrate"
	"github.com/nam3886/kratos-modular/internal/platform/migrate/gen"
)

func main() {
	gen.Main(migrate.Tables)
}
