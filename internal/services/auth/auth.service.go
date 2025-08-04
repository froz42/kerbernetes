package authsvc

import (
	"context"
	"errors"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	envsvc "github.com/froz42/kerbernetes/internal/services/env"
	k8ssvc "github.com/froz42/kerbernetes/internal/services/k8s"
	k8smodels "github.com/froz42/kerbernetes/internal/services/k8s/models"
	ldapsvc "github.com/froz42/kerbernetes/internal/services/ldap"
	"github.com/samber/do"
)

type AuthService interface {
	AuthAccount(ctx context.Context, username string) (*k8smodels.Credentials, error)
}

type authService struct {
	env     envsvc.Env
	k8sSvc  k8ssvc.K8sService
	ldapSvc ldapsvc.LDAPSvc
	logger  *slog.Logger
}

func NewProvider() func(i *do.Injector) (AuthService, error) {
	return func(i *do.Injector) (AuthService, error) {
		return New(
			do.MustInvoke[envsvc.EnvSvc](i),
			do.MustInvoke[k8ssvc.K8sService](i),
			do.MustInvoke[ldapsvc.LDAPSvc](i),
			do.MustInvoke[*slog.Logger](i),
		)
	}
}

func New(
	configService envsvc.EnvSvc,
	k8sService k8ssvc.K8sService,
	ldapsvc ldapsvc.LDAPSvc,
	logger *slog.Logger,
) (AuthService, error) {
	return &authService{
		env:     configService.GetEnv(),
		k8sSvc:  k8sService,
		ldapSvc: ldapsvc,
		logger:  logger.With("service", "auth"),
	}, nil
}

func (s *authService) AuthAccount(ctx context.Context, username string) (*k8smodels.Credentials, error) {
	s.logger.Info("Authenticating user", "username", username)
	// in case of LDAP we first try to get the user from LDAP
	if s.env.LDAPEnabled {
		user, err := s.ldapSvc.GetUser(username)
		if err != nil {
			s.logger.Error("Failed to get user from LDAP", "username", username, "error", err)
			return nil, huma.Error401Unauthorized("unauthorized", errors.New("failed to authenticate user on LDAP"))
		}
		groups, err := s.ldapSvc.GetUserGroups(user.DN)
		if err != nil {
			s.logger.Error("Failed to get user groups from LDAP", "username", username, "error", err)
			return nil, huma.Error401Unauthorized("Unauthorized", errors.New("failed to authenticate user on LDAP"))
		}
		s.logger.Info("User authenticated via LDAP", "username", username, "dn", user.DN, "groups", groups)
	}
	return nil, huma.Error501NotImplemented("Not Implemented")
}
