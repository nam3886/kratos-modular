package data

import (
	"context"

	"github.com/nam3886/kratos-modular/internal/module/payment/internal/biz"
	"github.com/nam3886/kratos-modular/internal/module/payment/internal/data/ent"
	"github.com/nam3886/kratos-modular/internal/platform/mq"
)

type limitChangeRepo struct {
	data *Data
}

func NewLimitChangeRepo(data *Data) biz.LimitChangeRepo {
	return &limitChangeRepo{data: data}
}

// Record stores the change once per message: the inbox entry and the row
// commit together, so a redelivered message changes nothing.
func (r *limitChangeRepo) Record(ctx context.Context, messageID, currency string, maxAmount int64) error {
	return r.data.inTx(ctx, func(tx *ent.Tx) error {
		first, err := mq.FirstTime(ctx, tx, Consumer, messageID)
		if err != nil || !first {
			return err
		}
		return tx.LimitChange.Create().SetCurrency(currency).SetMaxAmount(maxAmount).Exec(ctx)
	})
}
