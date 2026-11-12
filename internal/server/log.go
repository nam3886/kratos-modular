package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/middleware/logging"
	"github.com/go-kratos/kratos/v3/transport"
)

// maxArgsLogged caps the request text written to the log.
const maxArgsLogged = 2000

// requestLog writes one line per request: ERROR for server faults, WARN for
// client faults, INFO otherwise. A request type that carries secrets hides
// them by implementing logging.Redacter.
func requestLog(logger *slog.Logger) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			start := time.Now()
			reply, err := next(ctx, req)
			var kind, op string
			if tr, ok := transport.FromServerContext(ctx); ok {
				kind, op = tr.Kind().String(), tr.Operation()
			}
			code := 200
			level := slog.LevelInfo
			if se := errors.FromError(err); se != nil {
				code = int(se.Code)
				level = slog.LevelWarn
				if code >= 500 {
					level = slog.LevelError
				}
			}
			logger.LogAttrs(ctx, level, "request",
				slog.String("kind", kind),
				slog.String("operation", op),
				slog.String("args", args(req)),
				slog.Int("code", code),
				slog.String("reason", errors.Reason(err)),
				slog.Duration("latency", time.Since(start)),
			)
			return reply, err
		}
	}
}

func args(req any) string {
	var s string
	switch v := req.(type) {
	case logging.Redacter:
		s = v.Redact()
	case fmt.Stringer:
		s = v.String()
	default:
		s = fmt.Sprintf("%+v", req)
	}
	if len(s) > maxArgsLogged {
		s = s[:maxArgsLogged] + "..."
	}
	return s
}
