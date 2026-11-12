package server

import (
	"log/slog"
	stdhttp "net/http"

	"github.com/nam3886/kratos-modular/internal/conf"
	"github.com/nam3886/kratos-modular/internal/platform/db"

	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/http"
)

func NewHTTPServer(c *conf.Server, logger *slog.Logger, m *Metrics, dbs *db.Manager, modules []Registrar) *http.Server {
	var filters []http.FilterFunc
	// CORS goes first so that 413 responses from bodyLimit carry its headers.
	if len(c.GetHttp().GetCorsAllowedOrigins()) > 0 {
		filters = append(filters, cors(c.GetHttp().GetCorsAllowedOrigins()))
	}
	filters = append(filters, bodyLimit(c.GetHttp().GetMaxBodyBytes()))
	var opts = []http.ServerOption{
		http.Filter(filters...),
		// Without these the router falls back to http.DefaultServeMux, which
		// the Prometheus import fills with /debug/vars.
		http.NotFoundHandler(stdhttp.NotFoundHandler()),
		http.MethodNotAllowedHandler(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			stdhttp.Error(w, stdhttp.StatusText(stdhttp.StatusMethodNotAllowed), stdhttp.StatusMethodNotAllowed)
		})),
		http.Middleware(
			recovery.Recovery(),
			m.middleware,
			hideInternal,
		),
	}
	if c.GetHttp().GetNetwork() != "" {
		opts = append(opts, http.Network(c.GetHttp().GetNetwork()))
	}
	if c.GetHttp().GetAddr() != "" {
		opts = append(opts, http.Address(c.GetHttp().GetAddr()))
	}
	if c.GetHttp().GetTimeout() != nil {
		opts = append(opts, http.Timeout(c.GetHttp().GetTimeout().AsDuration()))
	}
	srv := http.NewServer(opts...)
	srv.Handle("/metrics", m.handler)
	srv.HandleFunc("/healthz", func(w stdhttp.ResponseWriter, _ *stdhttp.Request) { w.WriteHeader(stdhttp.StatusOK) })
	srv.HandleFunc("/readyz", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		for _, d := range dbs.All() {
			if err := d.Pool.PingContext(r.Context()); err != nil {
				logger.ErrorContext(r.Context(), "not ready", "database", d.Name, "error", err)
				stdhttp.Error(w, "database "+d.Name+" unreachable", stdhttp.StatusServiceUnavailable)
				return
			}
		}
		w.WriteHeader(stdhttp.StatusOK)
	})
	for _, m := range modules {
		m.RegisterHTTP(srv)
	}
	return srv
}
