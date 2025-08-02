package services

import (
	authservice "github.com/froz42/kerbernetes/internal/services/auth"
	configservice "github.com/froz42/kerbernetes/internal/services/config"
	"github.com/samber/do"
)

func InitServices(i *do.Injector) error {
	do.Provide(i, configservice.NewProvider())
	do.Provide(i, authservice.NewProvider())
	return nil
}
