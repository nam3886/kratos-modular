package main

import (
	settingsv1 "github.com/nam3886/kratos-modular/api/settings/v1"
	"github.com/nam3886/kratos-modular/internal/module/payment"
	"github.com/nam3886/kratos-modular/internal/module/settings"
	"github.com/nam3886/kratos-modular/internal/server"
)

// settingsAPI is what the payment module is given to call settings with.
func settingsAPI(s *settings.Module) settingsv1.SettingsServiceServer {
	return s.API()
}

// registrars lists the modules that serve an API.
func registrars(s *settings.Module, p *payment.Module) []server.Registrar {
	return []server.Registrar{s, p}
}

// consumers lists the modules that handle events.
func consumers(p *payment.Module) []server.Consumer {
	return []server.Consumer{p}
}
