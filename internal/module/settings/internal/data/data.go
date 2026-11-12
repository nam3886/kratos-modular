package data

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/wire"

	"github.com/nam3886/kratos-modular/internal/module/settings/internal/data/ent"
	"github.com/nam3886/kratos-modular/internal/platform/db"
)

// database is the name of the configured database this module stores into.
const database = "main"

var ProviderSet = wire.NewSet(NewData, NewSettingRepo)

// Data holds the module's ent client. The client runs on the shared pool of
// its database, so it is never closed here.
type Data struct {
	db *ent.Client
}

func NewData(dbs *db.Manager) (*Data, error) {
	d, err := dbs.Get(database)
	if err != nil {
		return nil, err
	}
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, d.Pool)))
	if d.Debug {
		client = client.Debug()
	}
	return &Data{db: client}, nil
}

// inTx runs fn in one transaction: commit when fn returns nil, roll back otherwise.
func (d *Data) inTx(ctx context.Context, fn func(tx *ent.Tx) error) error {
	tx, err := d.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if v := recover(); v != nil {
			_ = tx.Rollback()
			panic(v)
		}
	}()
	if err := fn(tx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			return errors.Join(err, fmt.Errorf("rollback: %w", rerr))
		}
		return err
	}
	return tx.Commit()
}
