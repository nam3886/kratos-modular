package server

import (
	"github.com/nam3886/kratos-modular/internal/conf"

	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/middleware/validate"
	"github.com/go-kratos/kratos/v3/transport/http"
	"go.einride.tech/aip/fieldbehavior"
	"google.golang.org/protobuf/proto"
)

// NewHTTPServer new an HTTP server, mounts every module on it and serves
// the metrics at /metrics.
func NewHTTPServer(c *conf.Server, m *Metrics, modules []Registrar) *http.Server {
	filters := []http.FilterFunc{bodyLimit(c.GetHttp().GetMaxBodyBytes())}
	if len(c.GetHttp().GetCorsAllowedOrigins()) > 0 {
		filters = append(filters, cors(c.GetHttp().GetCorsAllowedOrigins()))
	}
	var opts = []http.ServerOption{
		http.Filter(filters...),
		http.Middleware(
			recovery.Recovery(),
			m.middleware,
			validate.Validator(func(req any) error {
				if msg, ok := req.(proto.Message); ok {
					return fieldbehavior.ValidateRequiredFields(msg)
				}
				return nil
			}),
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
