package openapi

import (
	"github.com/danielgtaylor/huma/v2"
)

type additionalSpecs struct {
	Tags []*huma.Tag `yaml:"tags"`
}

func WithOverviewDoc(humaConfig huma.Config) huma.Config {
	humaConfig.Info.Contact = &huma.Contact{
		Email: "froz@theomatis.fr",
		Name:  "Froz",
	}
	return humaConfig
}
