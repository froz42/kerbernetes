package authsvc

import (
	"context"

	configsvc "github.com/froz42/kerbernetes/internal/services/config"
	k8ssvc "github.com/froz42/kerbernetes/internal/services/k8s"
	k8smodels "github.com/froz42/kerbernetes/internal/services/k8s/models"
	"github.com/samber/do"
)

type AuthService interface {
	AuthAccount(ctx context.Context, username string) (*k8smodels.Credentials, error)
}

type authService struct {
	configsvc configsvc.ConfigService
	k8sSvc    k8ssvc.K8sService
}

func NewProvider() func(i *do.Injector) (AuthService, error) {
	return func(i *do.Injector) (AuthService, error) {
		return New(
			do.MustInvoke[configsvc.ConfigService](i),
			do.MustInvoke[k8ssvc.K8sService](i),
		)
	}
}

func New(
	configService configsvc.ConfigService,
	k8sService k8ssvc.K8sService,
) (AuthService, error) {
	return &authService{
		configsvc: configService,
		k8sSvc:    k8sService,
	}, nil
}

func (s *authService) AuthAccount(ctx context.Context, username string) (*k8smodels.Credentials, error) {
	// Delegate the authentication to the K8s service
	return s.k8sSvc.AuthAccount(ctx, username)
}
