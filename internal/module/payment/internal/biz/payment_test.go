package biz

import (
	"context"
	"errors"
	"testing"
	"uuid"
)

type fakeRepo struct {
	PaymentRepo
	created []*Payment
}

func (r *fakeRepo) Create(_ context.Context, p *Payment) (*Payment, error) {
	r.created = append(r.created, p)
	return p, nil
}

type fakeLimits map[string]int64

func (l fakeLimits) MaxAmount(_ context.Context, currency string) (int64, error) {
	limit, ok := l[currency]
	if !ok {
		return 0, ErrCurrencyNotSupported
	}
	return limit, nil
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name     string
		amount   int64
		currency string
		want     error
	}{
		{"within limit", 500, "usd", nil},
		{"at limit", 1000, "USD", nil},
		{"over limit", 1001, "USD", ErrLimitExceeded},
		{"currency without setting", 5, "EUR", ErrCurrencyNotSupported},
		{"zero amount", 0, "USD", ErrPaymentAmountInvalid},
		{"bad currency", 5, "US", ErrPaymentCurrencyInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}
			uc := NewPaymentUsecase(repo, fakeLimits{"USD": 1000}, nil)
			p, err := uc.Create(context.Background(), tt.amount, tt.currency)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if tt.want != nil {
				if len(repo.created) != 0 {
					t.Fatal("a rejected payment was stored")
				}
				return
			}
			if p.Currency != "USD" || p.ID == uuid.Nil() || len(repo.created) != 1 {
				t.Fatalf("stored %+v, created=%d", p, len(repo.created))
			}
		})
	}
}
