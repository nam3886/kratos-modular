package server

import (
	"github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/wire"

	"github.com/nam3886/kratos-modular/internal/platform/mq"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer, NewMessaging, NewMetrics)

// Registrar is a module that serves an API.
type Registrar interface {
	RegisterHTTP(*http.Server)
	RegisterGRPC(*grpc.Server)
}

// Consumer is a module that handles events.
type Consumer interface {
	RegisterConsumers(*mq.MQ) error
}
