//go:build wireinject

package payment

import (
	"github.com/google/wire"

	settingsv1 "github.com/nam3886/kratos-modular/api/settings/v1"
	"github.com/nam3886/kratos-modular/internal/module/payment/internal/biz"
	"github.com/nam3886/kratos-modular/internal/module/payment/internal/data"
	"github.com/nam3886/kratos-modular/internal/module/payment/internal/service"
	"github.com/nam3886/kratos-modular/internal/platform/db"
)

// New assembles the module. settings is the settings module's API.
func New(*db.Manager, settingsv1.SettingsServiceServer) (*Module, error) {
	panic(wire.Build(data.ProviderSet, biz.ProviderSet, service.ProviderSet, newModule))
}
