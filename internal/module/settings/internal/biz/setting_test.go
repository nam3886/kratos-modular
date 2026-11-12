package biz

import (
	"context"
	"errors"
	"testing"
)

type fakeRepo struct {
	saved map[string]*Setting
}

func (r *fakeRepo) Find(_ context.Context, currency string) (*Setting, error) {
	s, ok := r.saved[currency]
	if !ok {
		return nil, ErrSettingNotFound
	}
	return s, nil
}

func (r *fakeRepo) Save(_ context.Context, s *Setting) (*Setting, error) {
	r.saved[s.Currency] = s
	return s, nil
}

func TestUpdateThenGet(t *testing.T) {
	uc := NewSettingUsecase(&fakeRepo{saved: map[string]*Setting{}})
	ctx := context.Background()

	if _, err := uc.Update(ctx, &Setting{Currency: "usd", MaxAmount: 1000}); err != nil {
		t.Fatal(err)
	}
	got, err := uc.Get(ctx, "Usd")
	if err != nil || got.Currency != "USD" || got.MaxAmount != 1000 {
		t.Fatalf("get = %+v, %v", got, err)
	}
	if _, err := uc.Get(ctx, "EUR"); !errors.Is(err, ErrSettingNotFound) {
		t.Fatalf("missing currency: %v", err)
	}
}

func TestUpdateRejectsInvalidSettings(t *testing.T) {
	uc := NewSettingUsecase(&fakeRepo{saved: map[string]*Setting{}})
	for _, c := range []struct {
		s    *Setting
		want error
	}{
		{&Setting{Currency: "US", MaxAmount: 10}, ErrSettingCurrencyInvalid},
		{&Setting{Currency: "USD", MaxAmount: 0}, ErrSettingMaxAmountInvalid},
		{&Setting{Currency: "USD", MaxAmount: -5}, ErrSettingMaxAmountInvalid},
	} {
		if _, err := uc.Update(context.Background(), c.s); !errors.Is(err, c.want) {
			t.Errorf("%+v: err = %v, want %v", c.s, err, c.want)
		}
	}
}
