package server

import (
	"github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/wire"

	"github.com/nam3886/kratos-modular/internal/platform/mq"
)

var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer, NewMessaging, NewMetrics)

type Registrar interface {
	RegisterHTTP(*http.Server)
	RegisterGRPC(*grpc.Server)
}

type Consumer interface {
	RegisterConsumers(*mq.MQ) error
}
