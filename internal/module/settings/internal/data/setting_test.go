package data

import (
	"context"
	"errors"
	"testing"

	"github.com/nam3886/kratos-modular/internal/module/settings/internal/biz"
	"github.com/nam3886/kratos-modular/internal/platform/testkit"
)

func TestSaveStoresSettingAndEventTogether(t *testing.T) {
	dbs := testkit.Postgres(t)
	d, err := NewData(dbs)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewSettingRepo(d)
	ctx := context.Background()

	if _, err := repo.Find(ctx, "USD"); !errors.Is(err, biz.ErrSettingNotFound) {
		t.Fatalf("missing setting: %v", err)
	}
	// Creating and then updating the same currency keeps one row and
	// publishes one event per change.
	for _, limit := range []int64{1000, 2000} {
		if _, err := repo.Save(ctx, &biz.Setting{Currency: "USD", MaxAmount: limit}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := repo.Find(ctx, "USD")
	if err != nil || got.MaxAmount != 2000 {
		t.Fatalf("find = %+v, %v", got, err)
	}
	if rows, events := testkit.Count(t, dbs, "settings"), testkit.Count(t, dbs, "watermill_outbox"); rows != 1 || events != 2 {
		t.Fatalf("settings=%d outbox=%d, want 1 and 2", rows, events)
	}
}
