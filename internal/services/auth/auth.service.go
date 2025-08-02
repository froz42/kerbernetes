package authservice

import (
	configservice "github.com/froz42/kerbernetes/internal/services/config"
	"github.com/samber/do"
)

type AuthService interface {
}

type authService struct {
	configService configservice.ConfigService
}

func NewProvider() func(i *do.Injector) (AuthService, error) {
	return func(i *do.Injector) (AuthService, error) {
		return New(
			do.MustInvoke[configservice.ConfigService](i),
		)
	}
}

func New(
	configService configservice.ConfigService,
) (AuthService, error) {
	return &authService{
		configService: configService,
	}, nil
}
