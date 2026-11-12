// Package db opens every configured database once and hands the shared
// connection pools to the modules.
package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/XSAM/otelsql"
	_ "github.com/jackc/pgx/v5/stdlib"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/nam3886/kratos-modular/internal/conf"
)

type Database struct {
	Name  string
	Pool  *sql.DB
	Debug bool
}

type Manager struct {
	dbs map[string]*Database
}

// Open opens a pool per configured database and verifies each is reachable.
func Open(c *conf.Data) (*Manager, func(), error) {
	m := &Manager{dbs: make(map[string]*Database)}
	cleanup := func() {
		for _, d := range m.dbs {
			_ = d.Pool.Close()
		}
	}
	for name, dc := range c.GetDatabases() {
		pool, err := otelsql.Open(dc.GetDriver(), dc.GetSource(), otelsql.WithAttributes(semconv.DBSystemPostgreSQL, semconv.DBNamespace(name)),
			// Only statements that belong to a request or a message are traced;
			// the outbox poll would otherwise add a trace every 200ms.
			otelsql.WithSpanOptions(otelsql.SpanOptions{
				OmitConnResetSession: true,
				OmitConnectorConnect: true,
				OmitRows:             true,
				SpanFilter: func(ctx context.Context, _ otelsql.Method, _ string, _ []driver.NamedValue) bool {
					return trace.SpanContextFromContext(ctx).IsValid()
				},
			}))
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("open database %q: %w", name, err)
		}
		if n := dc.GetMaxOpenConns(); n > 0 {
			pool.SetMaxOpenConns(int(n))
		}
		m.dbs[name] = &Database{Name: name, Pool: pool, Debug: dc.GetDebug()}
		if err := pool.PingContext(context.Background()); err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("ping database %q: %w", name, err)
		}
	}
	return m, cleanup, nil
}

func (m *Manager) Get(name string) (*Database, error) {
	d, ok := m.dbs[name]
	if !ok {
		return nil, fmt.Errorf("database %q is not configured", name)
	}
	return d, nil
}

func (m *Manager) All() []*Database {
	out := make([]*Database, 0, len(m.dbs))
	for _, d := range m.dbs {
		out = append(out, d)
	}
	return out
}
