package authsvc

import (
	configsvc "github.com/froz42/kerbernetes/internal/services/config"
	"github.com/samber/do"
)

type AuthService interface {
}

type authService struct {
	configsvc configsvc.ConfigService
}

func NewProvider() func(i *do.Injector) (AuthService, error) {
	return func(i *do.Injector) (AuthService, error) {
		return New(
			do.MustInvoke[configsvc.ConfigService](i),
		)
	}
}

func New(
	configService configsvc.ConfigService,
) (AuthService, error) {
	return &authService{
		configsvc: configService,
	}, nil
}
