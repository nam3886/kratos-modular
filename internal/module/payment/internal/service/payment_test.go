package service

import (
	"context"
	"testing"
	"uuid"

	v1 "github.com/nam3886/kratos-modular/api/payment/v1"
	"github.com/nam3886/kratos-modular/internal/module/payment/internal/biz"

	"github.com/go-kratos/kratos/v3/errors"
)

type memRepo struct {
	biz.PaymentRepo
	byID map[uuid.UUID]*biz.Payment
}

func (r *memRepo) Create(_ context.Context, p *biz.Payment) (*biz.Payment, error) {
	p.Status = "pending"
	r.byID[p.ID] = p
	return p, nil
}

func (r *memRepo) Find(_ context.Context, id uuid.UUID) (*biz.Payment, error) {
	p, ok := r.byID[id]
	if !ok {
		return nil, biz.ErrPaymentNotFound
	}
	return p, nil
}

type limits int64

func (l limits) MaxAmount(context.Context, string) (int64, error) { return int64(l), nil }

func newService() *PaymentService {
	return NewPaymentService(biz.NewPaymentUsecase(&memRepo{byID: map[uuid.UUID]*biz.Payment{}}, limits(1000), nil))
}

func TestCreateThenGet(t *testing.T) {
	s := newService()
	ctx := context.Background()
	created, err := s.CreatePayment(ctx, &v1.CreatePaymentRequest{Payment: &v1.Payment{Amount: 10, Currency: "usd"}})
	if err != nil {
		t.Fatal(err)
	}
	if created.GetCurrency() != "USD" || created.GetStatus() != "pending" {
		t.Fatalf("created = %v", created)
	}
	got, err := s.GetPayment(ctx, &v1.GetPaymentRequest{Id: created.GetId()})
	if err != nil || got.GetAmount() != 10 {
		t.Fatalf("get = %v, %v", got, err)
	}
}

func TestErrorsCarryTheAPIReason(t *testing.T) {
	s := newService()
	ctx := context.Background()
	_, err := s.CreatePayment(ctx, &v1.CreatePaymentRequest{Payment: &v1.Payment{Amount: 5000, Currency: "USD"}})
	if errors.Reason(err) != v1.ErrorReason_PAYMENT_LIMIT_EXCEEDED.String() || errors.Code(err) != 400 {
		t.Fatalf("over limit: reason=%q code=%d", errors.Reason(err), errors.Code(err))
	}
	_, err = s.GetPayment(ctx, &v1.GetPaymentRequest{Id: "not-a-uuid"})
	if errors.Reason(err) != v1.ErrorReason_PAYMENT_INVALID_ARGUMENT.String() {
		t.Fatalf("bad id: reason=%q", errors.Reason(err))
	}
	_, err = s.GetPayment(ctx, &v1.GetPaymentRequest{Id: uuid.NewV7().String()})
	if !errors.IsNotFound(err) {
		t.Fatalf("missing: %v", err)
	}
}
