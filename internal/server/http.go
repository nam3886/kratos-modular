package server

import (
	stdhttp "net/http"

	"github.com/nam3886/kratos-modular/internal/conf"

	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/http"
)

// NewHTTPServer new an HTTP server, mounts every module on it and serves
// the metrics at /metrics.
func NewHTTPServer(c *conf.Server, m *Metrics, modules []Registrar) *http.Server {
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
	for _, m := range modules {
		m.RegisterHTTP(srv)
	}
	return srv
}
