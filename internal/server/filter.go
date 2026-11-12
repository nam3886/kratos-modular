package server

import (
	nethttp "net/http"

	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/gorilla/handlers"
)

// defaultMaxBodyBytes applies when the config sets no limit.
const defaultMaxBodyBytes = 1 << 20

// cors lets browsers on the given origins call the API.
func cors(origins []string) http.FilterFunc {
	return handlers.CORS(
		handlers.AllowedOrigins(origins),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)
}

// bodyLimit rejects request bodies larger than limit bytes.
func bodyLimit(limit int64) http.FilterFunc {
	if limit <= 0 {
		limit = defaultMaxBodyBytes
	}
	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			if r.ContentLength > limit {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(nethttp.StatusRequestEntityTooLarge)
				_, _ = w.Write([]byte(`{"code":413,"reason":"REQUEST_TOO_LARGE","message":"request body is too large"}`))
				return
			}
			// A body without a declared length is cut off at the limit while it is read.
			r.Body = nethttp.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}
