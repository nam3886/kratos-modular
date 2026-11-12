package main

import (
	"github.com/nam3886/kratos-modular/internal/module/settings"
	"github.com/nam3886/kratos-modular/internal/server"
)

// registrars lists the modules that serve an API.
func registrars(s *settings.Module) []server.Registrar {
	return []server.Registrar{s}
}

// consumers lists the modules that handle events.
func consumers() []server.Consumer {
	return nil
}
