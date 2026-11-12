//go:build wireinject

package main

import (
	"log/slog"

	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"

	"github.com/nam3886/kratos-modular/internal/conf"
	"github.com/nam3886/kratos-modular/internal/module/payment"
	"github.com/nam3886/kratos-modular/internal/module/settings"
	"github.com/nam3886/kratos-modular/internal/platform/db"
	"github.com/nam3886/kratos-modular/internal/platform/mq"
	"github.com/nam3886/kratos-modular/internal/server"
)

func wireApp(*conf.Server, *conf.Data, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(
		db.Open,
		mq.New,
		settings.New,
		settingsAPI,
		payment.New,
		registrars,
		consumers,
		server.ProviderSet,
		newApp,
	))
}
