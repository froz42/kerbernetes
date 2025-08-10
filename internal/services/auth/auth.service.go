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
	v1 "github.com/froz42/kerbernetes/k8s/api/rbac.kerbernetes.io/v1"
	"github.com/samber/do"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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
		err := s.ldapReconcilate(ctx, username, sa)
		if err != nil {
			return nil, err
		}
	}

	token, err := s.serviceAccountsSvc.IssueToken(ctx, sa.Name)
	if err != nil {
		s.logger.Error("Failed to issue token", "username", username, "error", err)
		return nil, huma.Error500InternalServerError("Failed to issue token")
	}
	s.logger.Info("Token issued for user", "username", username)

	return &k8smodels.Credentials{
		Kind:       "ExecCredential",
		ApiVersion: "client.authentication.k8s.io/v1beta1",
		Status: &k8smodels.Status{
			Token:               token.Status.Token,
			ExpirationTimestamp: token.Status.ExpirationTimestamp.Time,
		},
	}, nil
}

func (s *authService) ldapReconcilate(ctx context.Context, username string, sa *corev1.ServiceAccount) error {
	user, err := s.ldapSvc.GetUser(username)
	if err != nil {
		s.logger.Error("Failed to get user from LDAP", "username", username, "error", err)
		return huma.Error401Unauthorized("failed to authenticate user on LDAP")
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
		return huma.Error401Unauthorized("failed to authenticate user on LDAP")
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
	groupsMap := make(map[string]bool)
	for _, group := range groups {
		groupsMap[group] = true
	}

	ldapBindings := s.ldapGroupBindingsSvc.GetBindings()
	// Filter bindings for the user
	var userBindings []*v1.LdapGroupBinding
	for _, binding := range ldapBindings {
		if _, exists := groupsMap[binding.Spec.LdapGroupDN]; exists {
			userBindings = append(userBindings, binding)
		}
	}

	// Reconcile cluster role bindings for the service account
	err = s.reconcileClusterAndRoleBindings(ctx, sa.Name, userBindings)
	if err != nil {
		s.logger.Error(
			"Failed to reconcile cluster role bindings",
			"username",
			username,
			"error",
			err,
		)
		return huma.Error500InternalServerError(
			"Failed to reconcile cluster role bindings",
		)
	}
	return nil
}

func (s *authService) reconcileClusterAndRoleBindings(
	ctx context.Context,
	saName string,
	ldapGroupBindings []*v1.LdapGroupBinding,
) error {
	s.logger.Info(
		"Starting reconciliation of ClusterRoleBindings and RoleBindings for ServiceAccount",
		"serviceAccount",
		saName,
	)

	// ------------------------------
	// 1. Retrieve current state
	// ------------------------------
	clusterRoleBindingsMap, err := s.getExistingClusterRoleBindings(ctx, saName)
	if err != nil {
		return err
	}

	roleBindingsMap, err := s.getExistingRoleBindings(ctx, saName)
	if err != nil {
		return err
	}

	// ------------------------------
	// 2. Ensure desired bindings exist
	// ------------------------------
	for _, ldapGroupBinding := range ldapGroupBindings {
		for _, binding := range ldapGroupBinding.Spec.Bindings {

			bindingName := serviceaccountssvc.GenBindingName(
				saName,
				binding.Name,
				ldapGroupBinding.Name,
			)

			switch binding.Kind {
			case "ClusterRole":
				err = s.ensureClusterRoleBinding(
					ctx,
					saName,
					ldapGroupBinding.Name,
					binding,
					bindingName,
					clusterRoleBindingsMap,
				)
				if err != nil {
					return err
				}

			case "Role":
				err = s.ensureRoleBinding(
					ctx,
					saName,
					ldapGroupBinding.Name,
					binding,
					bindingName,
					roleBindingsMap,
				)
				if err != nil {
					return err
				}

			default:
				s.logger.Warn(
					"Skipping unsupported binding kind",
					"serviceAccount", saName,
					"kind", binding.Kind,
					"name", binding.Name,
				)
			}
		}
	}

	// ------------------------------
	// 3. Remove bindings no longer needed
	// ------------------------------
	err = s.removeUnusedClusterRoleBindings(ctx, saName, clusterRoleBindingsMap)
	if err != nil {
		return err
	}

	err = s.removeUnusedRoleBindings(ctx, saName, roleBindingsMap)
	if err != nil {
		return err
	}

	s.logger.Info(
		"Completed reconciliation of ClusterRoleBindings and RoleBindings",
		"serviceAccount", saName,
		"remainingClusterRoleBindings", len(clusterRoleBindingsMap),
		"remainingRoleBindings", len(roleBindingsMap),
	)
	return nil
}

//
// -------- Helper functions --------
//

func (s *authService) getExistingClusterRoleBindings(
	ctx context.Context,
	saName string,
) (map[string]rbacv1.ClusterRoleBinding, error) {
	clusterRoleBindings, err := s.serviceAccountsSvc.GetClusterRoleBindings(ctx, saName)
	if err != nil {
		s.logger.Error(
			"Unable to retrieve current ClusterRoleBindings",
			"serviceAccount", saName,
			"error", err,
		)
		return nil, huma.Error500InternalServerError("failed to get cluster role bindings")
	}

	result := make(map[string]rbacv1.ClusterRoleBinding, len(clusterRoleBindings))
	for _, b := range clusterRoleBindings {
		result[b.Name] = b
	}
	return result, nil
}

func (s *authService) getExistingRoleBindings(
	ctx context.Context,
	saName string,
) (map[string]rbacv1.RoleBinding, error) {
	roleBindings, err := s.serviceAccountsSvc.GetRoleBindings(ctx, saName)
	if err != nil {
		s.logger.Error(
			"Unable to retrieve current RoleBindings",
			"serviceAccount", saName,
			"error", err,
		)
		return nil, huma.Error500InternalServerError("failed to get role bindings")
	}

	result := make(map[string]rbacv1.RoleBinding, len(roleBindings))
	for _, b := range roleBindings {
		result[b.Name] = b
	}
	return result, nil
}

func (s *authService) ensureClusterRoleBinding(
	ctx context.Context,
	saName, ldapGroupBindingName string,
	binding v1.LdapGroupBindingItem,
	bindingName string,
	existingMap map[string]rbacv1.ClusterRoleBinding,
) error {
	existing, exists := existingMap[bindingName]

	if !exists {
		newBinding, err := s.serviceAccountsSvc.CreateClusterRoleBinding(
			ctx,
			saName,
			binding.Name,
			ldapGroupBindingName,
		)
		if err != nil {
			s.logger.Error(
				"Failed to create missing ClusterRoleBinding",
				"serviceAccount", saName,
				"role", binding.Name,
				"error", err,
			)
			return huma.Error500InternalServerError("failed to create cluster role binding")
		}
		s.logger.Info(
			"Created new ClusterRoleBinding",
			"serviceAccount", saName,
			"bindingName", newBinding.Name,
		)
	} else {
		if existing.RoleRef.Name != binding.Name {
			_, err := s.serviceAccountsSvc.UpdateClusterRoleBinding(
				ctx,
				saName,
				binding.Name,
				ldapGroupBindingName,
			)
			if err != nil {
				s.logger.Error(
					"Failed to update ClusterRoleBinding to match desired state",
					"serviceAccount", saName,
					"role", binding.Name,
					"error", err,
				)
				return huma.Error500InternalServerError("failed to update cluster role binding")
			}
			s.logger.Info(
				"Updated ClusterRoleBinding to match desired state",
				"serviceAccount", saName,
				"bindingName", existing.Name,
			)
		}
	}

	// delete from existing map to track unused bindings
	delete(existingMap, bindingName)
	return nil
}

func (s *authService) ensureRoleBinding(
	ctx context.Context,
	saName, ldapGroupBindingName string,
	binding v1.LdapGroupBindingItem,
	bindingName string,
	existingMap map[string]rbacv1.RoleBinding,
) error {
	if binding.Namespace == "" {
		s.logger.Warn(
			"Skipping RoleBinding creation/update due to missing namespace",
			"serviceAccount", saName,
			"roleName", binding.Name,
		)
		return nil
	}

	existing, exists := existingMap[bindingName]
	roleRef := rbacv1.RoleRef{
		APIGroup: binding.ApiGroup,
		Kind:     binding.Kind,
		Name:     binding.Name,
	}

	if !exists {
		newBinding, err := s.serviceAccountsSvc.CreateRoleBinding(
			ctx,
			saName,
			binding.Namespace,
			ldapGroupBindingName,
			roleRef,
		)
		if err != nil {
			s.logger.Error(
				"Failed to create missing RoleBinding",
				"serviceAccount", saName,
				"role", binding.Name,
				"namespace", binding.Namespace,
				"error", err,
			)
			return huma.Error500InternalServerError("failed to create role binding")
		}
		s.logger.Info(
			"Created new RoleBinding",
			"serviceAccount", saName,
			"bindingName", newBinding.Name,
		)
	} else {
		if existing.RoleRef.Name != binding.Name ||
			existing.RoleRef.Kind != binding.Kind ||
			existing.RoleRef.APIGroup != binding.ApiGroup {
			_, err := s.serviceAccountsSvc.UpdateRoleBinding(
				ctx,
				saName,
				binding.Namespace,
				roleRef,
				ldapGroupBindingName,
			)
			if err != nil {
				s.logger.Error(
					"Failed to update RoleBinding to match desired state",
					"serviceAccount", saName,
					"role", binding.Name,
					"namespace", binding.Namespace,
					"error", err,
				)
				return huma.Error500InternalServerError("failed to update role binding")
			}
			s.logger.Info(
				"Updated RoleBinding to match desired state",
				"serviceAccount", saName,
				"bindingName", existing.Name,
			)
		}
	}

	// delete from existing map to track unused bindings
	delete(existingMap, bindingName)
	return nil
}

func (s *authService) removeUnusedClusterRoleBindings(
	ctx context.Context,
	saName string,
	bindings map[string]rbacv1.ClusterRoleBinding,
) error {
	for name, binding := range bindings {
		s.logger.Info(
			"Removing ClusterRoleBinding not present in desired state",
			"serviceAccount", saName,
			"name", name,
			"role", binding.Name,
		)
		err := s.serviceAccountsSvc.DeleteClusterRoleBinding(ctx, name)
		if err != nil {
			s.logger.Error(
				"Failed to remove unused ClusterRoleBinding",
				"serviceAccount", saName,
				"name", name,
				"error", err,
			)
			return huma.Error500InternalServerError("failed to delete cluster role binding")
		}
	}
	return nil
}

func (s *authService) removeUnusedRoleBindings(
	ctx context.Context,
	saName string,
	bindings map[string]rbacv1.RoleBinding,
) error {
	for name, binding := range bindings {
		s.logger.Info(
			"Removing RoleBinding not present in desired state",
			"serviceAccount", saName,
			"name", name,
			"role", binding.Name,
		)
		err := s.serviceAccountsSvc.DeleteRoleBinding(ctx, binding.Namespace, name)
		if err != nil {
			s.logger.Error(
				"Failed to remove unused RoleBinding",
				"serviceAccount", saName,
				"name", name,
				"error", err,
			)
			return huma.Error500InternalServerError("failed to delete role binding")
		}
	}
	return nil
}
