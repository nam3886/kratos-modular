package data

import (
	"context"
	"uuid"

	v1 "github.com/nam3886/kratos-modular/api/payment/v1"
	"github.com/nam3886/kratos-modular/internal/module/payment/internal/biz"
	"github.com/nam3886/kratos-modular/internal/module/payment/internal/data/ent"
	"github.com/nam3886/kratos-modular/internal/platform/mq"

	guuid "github.com/google/uuid"
)

type paymentRepo struct {
	data *Data
}

func NewPaymentRepo(data *Data) biz.PaymentRepo {
	return &paymentRepo{data: data}
}

// Create stores the payment and its PaymentCreated event in one transaction.
func (r *paymentRepo) Create(ctx context.Context, p *biz.Payment) (*biz.Payment, error) {
	var row *ent.Payment
	err := r.data.inTx(ctx, func(tx *ent.Tx) error {
		var err error
		row, err = tx.Payment.Create().SetID(guuid.UUID(p.ID)).SetAmount(p.Amount).SetCurrency(p.Currency).Save(ctx)
		if err != nil {
			return err
		}
		return mq.Publish(ctx, tx, uuid.NewV7().String(), &v1.PaymentCreated{Id: p.ID.String(), Amount: p.Amount, Currency: p.Currency})
	})
	if err != nil {
		return nil, err
	}
	return toBiz(row), nil
}

func (r *paymentRepo) Find(ctx context.Context, id uuid.UUID) (*biz.Payment, error) {
	row, err := r.data.db.Payment.Get(ctx, guuid.UUID(id))
	if ent.IsNotFound(err) {
		return nil, biz.ErrPaymentNotFound
	}
	if err != nil {
		return nil, err
	}
	return toBiz(row), nil
}

func toBiz(row *ent.Payment) *biz.Payment {
	return &biz.Payment{ID: uuid.UUID(row.ID), Amount: row.Amount, Currency: row.Currency, Status: row.Status}
}
