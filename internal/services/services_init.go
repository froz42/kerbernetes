package services

import (
	authsvc "github.com/froz42/kerbernetes/internal/services/auth"
	configsvc "github.com/froz42/kerbernetes/internal/services/config"
	k8ssvc "github.com/froz42/kerbernetes/internal/services/k8s"
	"github.com/samber/do"
)

func InitServices(i *do.Injector) error {
	do.Provide(i, configsvc.NewProvider())
	do.Provide(i, authsvc.NewProvider())
	do.Provide(i, k8ssvc.NewProvider())
	return nil
}
