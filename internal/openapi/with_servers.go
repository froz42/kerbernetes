package openapi

import (
	"github.com/danielgtaylor/huma/v2"
	envsvc "github.com/froz42/kerbernetes/internal/services/env"
)

func WithServers(humaConfig huma.Config, config envsvc.Env) huma.Config {
	humaConfig.Servers = []*huma.Server{
		{
			URL:         config.APIPrefix,
			Description: "Current Server",
		},
	}
	return humaConfig
}
