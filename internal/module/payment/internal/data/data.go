package data

import (
	"context"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/wire"

	"github.com/nam3886/kratos-modular/internal/module/payment/internal/data/ent"
	"github.com/nam3886/kratos-modular/internal/platform/db"
)

// database is the name of the configured database this module stores into.
const database = "main"

// Consumer names this module as an event consumer: it is the AMQP queue
// suffix and the inbox dedupe key. Renaming it re-delivers and re-processes
// old messages.
const Consumer = "payment"

var ProviderSet = wire.NewSet(NewData, NewPaymentRepo, NewLimitChangeRepo, NewLimitReader)

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

func (d *Data) inTx(ctx context.Context, fn func(tx *ent.Tx) error) error {
	tx, err := d.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
