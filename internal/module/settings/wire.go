//go:build wireinject

package settings

import (
	"github.com/google/wire"

	"github.com/nam3886/kratos-modular/internal/module/settings/internal/biz"
	"github.com/nam3886/kratos-modular/internal/module/settings/internal/data"
	"github.com/nam3886/kratos-modular/internal/module/settings/internal/service"
	"github.com/nam3886/kratos-modular/internal/platform/db"
)

// New assembles the module.
func New(*db.Manager) (*Module, error) {
	panic(wire.Build(data.ProviderSet, biz.ProviderSet, service.ProviderSet, newModule))
}
