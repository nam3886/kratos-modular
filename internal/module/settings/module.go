// Package settings is the payment-settings module. Other modules may use only
// this package and api/settings/v1.
package settings

import (
	"github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/go-kratos/kratos/v3/transport/http"

	v1 "github.com/nam3886/kratos-modular/api/settings/v1"
	"github.com/nam3886/kratos-modular/internal/module/settings/internal/service"
)

type Module struct {
	svc *service.SettingsService
}

func newModule(svc *service.SettingsService) *Module {
	return &Module{svc: svc}
}

// API is what other modules call, in process.
func (m *Module) API() v1.SettingsServiceServer { return m.svc }

func (m *Module) RegisterHTTP(srv *http.Server) { v1.RegisterSettingsServiceHTTPServer(srv, m.svc) }

func (m *Module) RegisterGRPC(srv *grpc.Server) { v1.RegisterSettingsServiceServer(srv, m.svc) }
