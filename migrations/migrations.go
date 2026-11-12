// Package migrations embeds the SQL migrations: one directory per configured
// database, named after it.
package migrations

import "embed"

// FS holds every database's migration files.
//
//go:embed */*.sql
var FS embed.FS
