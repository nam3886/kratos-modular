package server

import (
	"context"
	"log/slog"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
)

// hideInternal logs any error that is not a Kratos error and replies with a
// generic one, so driver and library text never reaches clients.
func hideInternal(next middleware.Handler) middleware.Handler {
	return func(ctx context.Context, req any) (any, error) {
		reply, err := next(ctx, req)
		if err == nil {
			return reply, nil
		}
		var ke *errors.Error
		if errors.As(err, &ke) {
			return reply, err
		}
		var op string
		if tr, ok := transport.FromServerContext(ctx); ok {
			op = tr.Operation()
		}
		slog.ErrorContext(ctx, "internal error", "operation", op, "error", err)
		return nil, errors.InternalServer("INTERNAL", "internal error")
	}
}
