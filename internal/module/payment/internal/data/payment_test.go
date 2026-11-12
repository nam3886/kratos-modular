package data

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/nam3886/kratos-modular/internal/module/payment/internal/biz"
	"github.com/nam3886/kratos-modular/internal/platform/testkit"
)

func TestCreateStoresPaymentAndEventTogether(t *testing.T) {
	dbs := testkit.Postgres(t)
	d, err := NewData(dbs)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPaymentRepo(d)
	ctx := context.Background()
	p := &biz.Payment{ID: uuid.NewV7(), Amount: 42, Currency: "USD"}

	created, err := repo.Create(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != "pending" {
		t.Fatalf("status = %q", created.Status)
	}
	if rows, events := testkit.Count(t, dbs, "payments"), testkit.Count(t, dbs, "watermill_outbox"); rows != 1 || events != 1 {
		t.Fatalf("payments=%d outbox=%d, want 1 and 1", rows, events)
	}

	// The same ID again fails, and the failed attempt leaves no event behind.
	if _, err := repo.Create(ctx, p); err == nil {
		t.Fatal("duplicate payment was accepted")
	}
	if rows, events := testkit.Count(t, dbs, "payments"), testkit.Count(t, dbs, "watermill_outbox"); rows != 1 || events != 1 {
		t.Fatalf("after failed create: payments=%d outbox=%d, want 1 and 1", rows, events)
	}

	got, err := repo.Find(ctx, p.ID)
	if err != nil || got.Amount != 42 || got.ID != p.ID {
		t.Fatalf("find = %+v, %v", got, err)
	}
	if _, err := repo.Find(ctx, uuid.NewV7()); !errors.Is(err, biz.ErrPaymentNotFound) {
		t.Fatalf("missing payment: %v", err)
	}
}

func TestRecordIsIdempotentPerMessage(t *testing.T) {
	dbs := testkit.Postgres(t)
	d, err := NewData(dbs)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewLimitChangeRepo(d)
	ctx := context.Background()
	for range 3 {
		if err := repo.Record(ctx, "msg-1", "USD", 1000); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Record(ctx, "msg-2", "USD", 2000); err != nil {
		t.Fatal(err)
	}
	if n := testkit.Count(t, dbs, "limit_changes"); n != 2 {
		t.Fatalf("limit_changes = %d, want 2", n)
	}
}
