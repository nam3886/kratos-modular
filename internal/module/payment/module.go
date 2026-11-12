// Package payment is the payment module. Other modules may use only this
// package and api/payment/v1.
package payment

import (
	"github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/go-kratos/kratos/v3/transport/http"

	v1 "github.com/nam3886/kratos-modular/api/payment/v1"
	settingsv1 "github.com/nam3886/kratos-modular/api/settings/v1"
	"github.com/nam3886/kratos-modular/internal/module/payment/internal/service"
	"github.com/nam3886/kratos-modular/internal/platform/mq"
)

// Module is the assembled payment module.
type Module struct {
	svc    *service.PaymentService
	events *service.SettingEvents
}

func newModule(svc *service.PaymentService, events *service.SettingEvents) *Module {
	return &Module{svc: svc, events: events}
}

// RegisterHTTP mounts the module's HTTP routes.
func (m *Module) RegisterHTTP(srv *http.Server) { v1.RegisterPaymentServiceHTTPServer(srv, m.svc) }

// RegisterGRPC mounts the module's gRPC service.
func (m *Module) RegisterGRPC(srv *grpc.Server) { v1.RegisterPaymentServiceServer(srv, m.svc) }

// RegisterConsumers subscribes the module to the events it handles.
func (m *Module) RegisterConsumers(q *mq.MQ) error {
	return q.Handle("payment", mq.Topic(&settingsv1.SettingChanged{}), m.events.OnSettingChanged)
}
