// Package migrations embeds the SQL migrations: one directory per configured
// database, named after it.
package migrations

import "embed"

//go:embed */*.sql
var FS embed.FS
