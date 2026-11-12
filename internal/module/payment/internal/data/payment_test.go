package data

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/nam3886/kratos-modular/internal/module/payment/internal/biz"
	"github.com/nam3886/kratos-modular/internal/platform/testkit"

	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
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

func TestListFiltersOrdersAndPages(t *testing.T) {
	d, err := NewData(testkit.Postgres(t))
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPaymentRepo(d)
	ctx := context.Background()
	for _, p := range []*biz.Payment{{Amount: 10, Currency: "USD"}, {Amount: 20, Currency: "USD"}, {Amount: 30, Currency: "USD"}, {Amount: 5, Currency: "EUR"}} {
		p.ID = uuid.NewV7()
		if _, err := repo.Create(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	decl, _ := filtering.NewDeclarations(filtering.DeclareStandardFunctions(), filtering.DeclareIdent("currency", filtering.TypeString))
	filter, err := filtering.ParseFilter(listRequest{filter: `currency="USD"`}, decl)
	if err != nil {
		t.Fatal(err)
	}
	orderBy, err := ordering.ParseOrderBy(listRequest{orderBy: "amount desc"})
	if err != nil {
		t.Fatal(err)
	}

	got, total, err := repo.List(ctx, biz.ListQuery{Filter: filter, OrderBy: orderBy, Offset: 2, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(got) != 1 || got[0].Amount != 10 {
		t.Fatalf("total=%d page=%+v, want total 3 and the 10 USD payment", total, got)
	}

	// Without an order the newest payment comes first.
	got, total, err = repo.List(ctx, biz.ListQuery{Limit: 1})
	if err != nil || total != 4 || got[0].Currency != "EUR" {
		t.Fatalf("default order: total=%d page=%+v err=%v", total, got, err)
	}

	bad, err := filtering.ParseFilter(listRequest{filter: `1=1`}, decl)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.List(ctx, biz.ListQuery{Filter: bad, Limit: 1}); !errors.Is(err, biz.ErrPaymentFilterInvalid) {
		t.Fatalf("1=1: err=%v, want ErrPaymentFilterInvalid", err)
	}
}

type listRequest struct{ filter, orderBy string }

func (r listRequest) GetFilter() string  { return r.filter }
func (r listRequest) GetOrderBy() string { return r.orderBy }
