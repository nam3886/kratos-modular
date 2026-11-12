package server

import (
	"context"
	"errors"
	"testing"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

func TestHideInternal(t *testing.T) {
	call := func(err error) error {
		_, got := hideInternal(func(context.Context, any) (any, error) { return nil, err })(context.Background(), nil)
		return got
	}

	got := call(errors.New(`pq: duplicate key value violates unique constraint "settings_pkey"`))
	if kerrors.Reason(got) != "INTERNAL" || kerrors.Code(got) != 500 || got.Error() != kerrors.InternalServer("INTERNAL", "internal error").Error() {
		t.Errorf("plain error: %v", got)
	}

	want := kerrors.BadRequest("PAYMENT_LIMIT_EXCEEDED", "over limit")
	if got := call(want); !errors.Is(got, want) || got.Error() != want.Error() {
		t.Errorf("kratos error: got %v, want it unchanged", got)
	}
}

func TestHideInternalClientClosed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, got := hideInternal(func(ctx context.Context, _ any) (any, error) { return nil, ctx.Err() })(ctx, nil)
	if kerrors.Code(got) != 499 || kerrors.Reason(got) != "CLIENT_CLOSED" {
		t.Errorf("cancelled request: %v", got)
	}
}
