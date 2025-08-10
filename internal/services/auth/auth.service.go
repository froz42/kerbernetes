package authsvc

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	envsvc "github.com/froz42/kerbernetes/internal/services/env"
	ldapgroupbindingssvc "github.com/froz42/kerbernetes/internal/services/k8s/ldapgroupbindings"
	k8smodels "github.com/froz42/kerbernetes/internal/services/k8s/models"
	serviceaccountssvc "github.com/froz42/kerbernetes/internal/services/k8s/serviceaccounts"
	ldapsvc "github.com/froz42/kerbernetes/internal/services/ldap"
	"github.com/samber/do"
)

type AuthService interface {
	AuthAccount(ctx context.Context, username string) (*k8smodels.Credentials, error)
}

type authService struct {
	env                  envsvc.Env
	serviceAccountsSvc   serviceaccountssvc.ServiceAccountsService
	ldapGroupBindingsSvc ldapgroupbindingssvc.LdapGroupBindingService
	ldapSvc              ldapsvc.LDAPSvc
	logger               *slog.Logger
}

func NewProvider() func(i *do.Injector) (AuthService, error) {
	return func(i *do.Injector) (AuthService, error) {
		return New(
			do.MustInvoke[envsvc.EnvSvc](i),
			do.MustInvoke[serviceaccountssvc.ServiceAccountsService](i),
			do.MustInvoke[ldapgroupbindingssvc.LdapGroupBindingService](i),
			do.MustInvoke[ldapsvc.LDAPSvc](i),
			do.MustInvoke[*slog.Logger](i),
		)
	}
}

func New(
	configService envsvc.EnvSvc,
	serviceAccountsSvc serviceaccountssvc.ServiceAccountsService,
	ldapGroupBindingsSvc ldapgroupbindingssvc.LdapGroupBindingService,
	ldapSvc ldapsvc.LDAPSvc,
	logger *slog.Logger,
) (AuthService, error) {
	return &authService{
		env:                  configService.GetEnv(),
		serviceAccountsSvc:   serviceAccountsSvc,
		ldapGroupBindingsSvc: ldapGroupBindingsSvc,
		ldapSvc:              ldapSvc,
		logger:               logger.With("service", "auth"),
	}, nil
}

func (s *authService) AuthAccount(
	ctx context.Context,
	username string,
) (*k8smodels.Credentials, error) {
	s.logger.Info("Authenticating user", "username", username)
	sa, err := s.serviceAccountsSvc.UpsertServiceAccount(ctx, username)
	if err != nil {
		s.logger.Error("Failed to upsert service account", "username", username, "error", err)
		return nil, huma.Error500InternalServerError("Failed to upsert service account")
	}

	// in case of LDAP we first try to get the user from LDAP
	if s.env.LDAPEnabled {
		user, err := s.ldapSvc.GetUser(username)
		if err != nil {
			s.logger.Error("Failed to get user from LDAP", "username", username, "error", err)
			return nil, huma.Error401Unauthorized("failed to authenticate user on LDAP")
		}
		groups, err := s.ldapSvc.GetUserGroups(user.DN)
		if err != nil {
			s.logger.Error(
				"Failed to get user groups from LDAP",
				"username",
				username,
				"error",
				err,
			)
			return nil, huma.Error401Unauthorized("failed to authenticate user on LDAP")
		}
		s.logger.Info(
			"User authenticated via LDAP",
			"username",
			username,
			"dn",
			user.DN,
			"groups",
			groups,
		)

		// Reconcile cluster role bindings for the service account
		err = s.reconciateClusterRoleBindings(ctx, sa.Name, groups)
		if err != nil {
			s.logger.Error(
				"Failed to reconcile cluster role bindings",
				"username",
				username,
				"error",
				err,
			)
			return nil, huma.Error500InternalServerError(
				"Failed to reconcile cluster role bindings",
			)
		}
	}

	return nil, huma.Error501NotImplemented("Not Implemented")
}

func (s *authService) reconciateClusterRoleBindings(ctx context.Context, saName string, groups []string) error {
	s.logger.Info("Reconciling cluster role bindings for user", "username", saName)
	bindings, err := s.serviceAccountsSvc.GetClusterRoleBindings(ctx, saName)
	if err != nil {
		s.logger.Error(
			"Failed to get cluster role bindings for user",
			"username",
			saName,
			"error",
			err,
		)
		return huma.Error500InternalServerError("Failed to get cluster role bindings")
	}
	for _, binding := range bindings {
		s.logger.Info(
			"Found cluster role binding",
			"name",
			binding.Name,
			"role",
			binding.RoleRef.Name,
		)
	}
	return nil
}
