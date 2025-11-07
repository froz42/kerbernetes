package openapi

import (
	"github.com/danielgtaylor/huma/v2"
)

func WithOverviewDoc(humaConfig huma.Config) huma.Config {
	humaConfig.Info.Contact = &huma.Contact{
		Email: "froz@theomatis.fr",
		Name:  "Froz",
	}
	return humaConfig
}
