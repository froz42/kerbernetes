package openapi

import (
	"os"

	"github.com/danielgtaylor/huma/v2"
	"gopkg.in/yaml.v3"
)

type additionalSpecs struct {
	Tags []*huma.Tag `yaml:"tags"`
}

func WithOverviewDoc(humaConfig huma.Config) huma.Config {
	humaConfig.Info.Contact = &huma.Contact{
		Email: "froz@theomatis.fr",
		Name:  "Froz",
	}
	doc, err := os.ReadFile("docs/API_DOC.md")
	if err != nil {
		return humaConfig
	}
	humaConfig.Info.Description = string(doc)

	yamlFile, err := os.ReadFile("docs/additional_specs.yaml")
	if err != nil {
		return humaConfig
	}
	additionalSpecs := additionalSpecs{}
	err = yaml.Unmarshal(yamlFile, &additionalSpecs)
	humaConfig.Tags = additionalSpecs.Tags
	if err != nil {
		return humaConfig
	}
	return humaConfig
}
