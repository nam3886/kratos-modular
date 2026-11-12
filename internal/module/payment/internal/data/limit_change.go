package data

import (
	"context"

	"github.com/nam3886/kratos-modular/internal/module/payment/internal/biz"
	"github.com/nam3886/kratos-modular/internal/module/payment/internal/data/ent"
	"github.com/nam3886/kratos-modular/internal/platform/mq"
)

// consumer is this module's name in the inbox and its queue suffix.
const consumer = "payment"

type limitChangeRepo struct {
	data *Data
}

// NewLimitChangeRepo returns the ent-backed biz.LimitChangeRepo.
func NewLimitChangeRepo(data *Data) biz.LimitChangeRepo {
	return &limitChangeRepo{data: data}
}

// Record stores the change once per message: the inbox entry and the row
// commit together, so a redelivered message changes nothing.
func (r *limitChangeRepo) Record(ctx context.Context, messageID, currency string, maxAmount int64) error {
	return r.data.inTx(ctx, func(tx *ent.Tx) error {
		first, err := mq.FirstTime(ctx, tx, consumer, messageID)
		if err != nil || !first {
			return err
		}
		return tx.LimitChange.Create().SetCurrency(currency).SetMaxAmount(maxAmount).Exec(ctx)
	})
}
