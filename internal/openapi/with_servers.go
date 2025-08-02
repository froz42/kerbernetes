package openapi

import (
	"github.com/danielgtaylor/huma/v2"
	configservice "github.com/froz42/kerbernetes/internal/services/config"
)

func WithServers(humaConfig huma.Config, config configservice.Config) huma.Config {
	humaConfig.Servers = []*huma.Server{
		{
			URL:         config.APIPrefix,
			Description: "Current Server",
		},
	}
	return humaConfig
}
