package serviceaccountssvc

import (
	"context"
	"fmt"
	"log/slog"

	envsvc "github.com/froz42/kerbernetes/internal/services/env"
	k8ssvc "github.com/froz42/kerbernetes/internal/services/k8s"
	"github.com/samber/do"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const saManagedLabel = "kerbernetes.io/managed"

type ServiceAccountsService interface {
	// UpsertServiceAccount retrieves or creates a service account for the given username
	UpsertServiceAccount(ctx context.Context, username string) (*corev1.ServiceAccount, error)

	// IssueToken creates a token for the service account
	IssueToken(ctx context.Context, username string) (*authv1.TokenRequest, error)

	// GetSAClusterRoleBindings get the cluster role bindings for an service account
	GetClusterRoleBindings(
		ctx context.Context,
		username string,
	) ([]rbacv1.ClusterRoleBinding, error)

	// CreateClusterRoleBinding creates a cluster role binding for the service account
	CreateClusterRoleBinding(
		ctx context.Context,
		username string,
		clusterRoleName string,
		ldapGroundBindingName string,
	) (*rbacv1.ClusterRoleBinding, error)

	UpdateClusterRoleBinding(
		ctx context.Context,
		username string,
		clusterRoleName string,
		ldapGroundBindingName string,
	) (*rbacv1.ClusterRoleBinding, error)

	// DeleteClusterRoleBinding deletes a cluster role binding by its name
	DeleteClusterRoleBinding(ctx context.Context, name string) error

	// GetRolesBindings retrieves the cluster role bindings for a service account
	GetRoleBindings(
		ctx context.Context,
		username string,
	) ([]rbacv1.RoleBinding, error)

	// CreateRoleBinding creates a role binding for the service account
	CreateRoleBinding(
		ctx context.Context,
		username string,
		namespace string,
		ldapGroundBindingName string,
		roleRef rbacv1.RoleRef,
	) (*rbacv1.RoleBinding, error)

	// UpdateRoleBinding updates an existing role binding for the service account
	UpdateRoleBinding(
		ctx context.Context,
		username string,
		namespace string,
		roleRef rbacv1.RoleRef,
		ldapGroundBindingName string,
	) (*rbacv1.RoleBinding, error)

	// DeleteRoleBinding deletes a role binding by its name
	DeleteRoleBinding(ctx context.Context, namespace string, name string) error
}

type serviceAccountsService struct {
	env       envsvc.Env
	clientset *kubernetes.Clientset
	namespace string
	logger    *slog.Logger
}

func NewProvider() func(i *do.Injector) (ServiceAccountsService, error) {
	return func(i *do.Injector) (ServiceAccountsService, error) {
		return New(
			do.MustInvoke[envsvc.EnvSvc](i).GetEnv(),
			do.MustInvoke[k8ssvc.K8sService](i),
			do.MustInvoke[*slog.Logger](i),
		)
	}
}

func New(
	env envsvc.Env,
	k8sSvc k8ssvc.K8sService,
	logger *slog.Logger,
) (ServiceAccountsService, error) {
	return &serviceAccountsService{
		env:       env,
		clientset: k8sSvc.GetClientset(),
		namespace: k8sSvc.GetNamespace(),
		logger:    logger.With("service", "serviceaccounts"),
	}, nil
}

// UpsertServiceAccount retrieves or creates a service account for the given username.
func (svc *serviceAccountsService) UpsertServiceAccount(
	ctx context.Context,
	username string,
) (*corev1.ServiceAccount, error) {
	sa, err := svc.clientset.CoreV1().
		ServiceAccounts(svc.namespace).
		Get(ctx, username, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return svc.createServiceAccount(ctx, username)
		}
		svc.logger.Error("Failed to get service account", "error", err)
		return nil, err
	}

	svc.logger.Info("Found existing service account", "name", sa.Name, "namespace", sa.Namespace)
	return sa, nil
}

// IssueToken creates a token for the service account.
func (svc *serviceAccountsService) IssueToken(
	ctx context.Context,
	username string,
) (*authv1.TokenRequest, error) {
	token, err := svc.clientset.CoreV1().ServiceAccounts(svc.namespace).
		CreateToken(ctx, username, &authv1.TokenRequest{
			Spec: authv1.TokenRequestSpec{
				Audiences:         []string{svc.env.TokenAudience},
				ExpirationSeconds: int64Ptr(int64(svc.env.TokenDuration)),
			},
		}, metav1.CreateOptions{})
	if err != nil {
		svc.logger.Error("Failed to create token for service account", "error", err)
		return nil, err
	}

	svc.logger.Info(
		"Issued token for service account",
		"name",
		username,
		"namespace",
		svc.namespace,
	)
	return token, nil
}

// GetClusterRoleBindings retrieves the cluster role bindings for a service account.
func (svc *serviceAccountsService) GetClusterRoleBindings(
	ctx context.Context,
	username string,
) ([]rbacv1.ClusterRoleBinding, error) {
	bindings, err := svc.clientset.RbacV1().
		ClusterRoleBindings().
		List(ctx, metav1.ListOptions{
			// we only want bindings that are managed by kerbernetes
			LabelSelector: saManagedLabel + "=true",
		})
	if err != nil {
		svc.logger.Error("Failed to get cluster role bindings", "error", err)
		return nil, err
	}
	// filter bindings for the specific service account
	var filteredBindings []rbacv1.ClusterRoleBinding
	for _, binding := range bindings.Items {
		for _, subject := range binding.Subjects {
			if subject.Kind == "ServiceAccount" && subject.Name == username &&
				subject.Namespace == svc.namespace {
				filteredBindings = append(filteredBindings, binding)
				break
			}
		}
	}

	svc.logger.Info(
		"Retrieved cluster role bindings",
		"username",
		username,
		"count",
		len(bindings.Items),
	)
	return bindings.Items, nil
}

// CreateClusterRoleBinding creates a cluster role binding for the service account.
func (svc *serviceAccountsService) CreateClusterRoleBinding(
	ctx context.Context,
	username string,
	roleName string,
	ldapGroundBindingName string,
) (*rbacv1.ClusterRoleBinding, error) {
	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: GenBindingName(username, roleName, ldapGroundBindingName),
			Labels: map[string]string{
				saManagedLabel: "true",
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      username,
				Namespace: svc.namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     roleName,
		},
	}

	binding, err := svc.clientset.RbacV1().
		ClusterRoleBindings().
		Create(ctx, binding, metav1.CreateOptions{})
	if err != nil {
		svc.logger.Error("Failed to create cluster role binding", "error", err)
		return nil, err
	}

	svc.logger.Info("Created cluster role binding", "name", binding.Name)
	return binding, nil
}

// UpdateClusterRoleBinding updates an existing cluster role binding for the service account.
func (svc *serviceAccountsService) UpdateClusterRoleBinding(
	ctx context.Context,
	username string,
	clusterRoleName string,
	ldapGroundBindingName string,
) (*rbacv1.ClusterRoleBinding, error) {
	name := GenBindingName(username, clusterRoleName, ldapGroundBindingName)
	binding, err := svc.clientset.RbacV1().
		ClusterRoleBindings().
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		svc.logger.Error("Failed to get cluster role binding", "error", err)
		return nil, err
	}

	binding.RoleRef.Name = clusterRoleName
	binding, err = svc.clientset.RbacV1().
		ClusterRoleBindings().
		Update(ctx, binding, metav1.UpdateOptions{})
	if err != nil {
		svc.logger.Error("Failed to update cluster role binding", "error", err)
		return nil, err
	}

	svc.logger.Info("Updated cluster role binding", "name", binding.Name)
	return binding, nil
}

// DeleteClusterRoleBinding deletes a cluster role binding by its name.
func (svc *serviceAccountsService) DeleteClusterRoleBinding(
	ctx context.Context,
	name string,
) error {
	err := svc.clientset.RbacV1().
		ClusterRoleBindings().
		Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		svc.logger.Error("Failed to delete cluster role binding", "error", err)
		return err
	}

	svc.logger.Info("Deleted cluster role binding", "name", name)
	return nil
}

// GetRoleBindings retrieves the role bindings accross all namespaces for a service account.
func (svc *serviceAccountsService) GetRoleBindings(
	ctx context.Context,
	username string,
) ([]rbacv1.RoleBinding, error) {
	bindings, err := svc.clientset.RbacV1().
		RoleBindings("").
		List(ctx, metav1.ListOptions{
			LabelSelector: saManagedLabel + "=true",
		})
	if err != nil {
		svc.logger.Error("Failed to get role bindings", "error", err)
		return nil, err
	}

	var filteredBindings []rbacv1.RoleBinding
	for _, binding := range bindings.Items {
		for _, subject := range binding.Subjects {
			if subject.Kind == "ServiceAccount" && subject.Name == username &&
				subject.Namespace == svc.namespace {
				filteredBindings = append(filteredBindings, binding)
				break
			}
		}
	}

	svc.logger.Info(
		"Retrieved role bindings",
		"username",
		username,
		"count",
		len(filteredBindings),
	)
	return filteredBindings, nil
}

// CreateRoleBinding creates a role binding for the service account.
func (svc *serviceAccountsService) CreateRoleBinding(
	ctx context.Context,
	username string,
	namespace string,
	ldapGroundBindingName string,
	roleRef rbacv1.RoleRef,
) (*rbacv1.RoleBinding, error) {
	name := GenBindingName(username, roleRef.Name, ldapGroundBindingName)
	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				saManagedLabel: "true",
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      username,
				Namespace: namespace,
			},
		},
		RoleRef: roleRef,
	}

	binding, err := svc.clientset.RbacV1().
		RoleBindings(namespace).
		Create(ctx, binding, metav1.CreateOptions{})
	if err != nil {
		svc.logger.Error("Failed to create role binding", "error", err)
		return nil, err
	}

	svc.logger.Info("Created role binding", "name", binding.Name, "namespace", namespace)
	return binding, nil
}

// UpdateRoleBinding updates an existing role binding for the service account.
func (svc *serviceAccountsService) UpdateRoleBinding(
	ctx context.Context,
	username string,
	namespace string,
	roleRef rbacv1.RoleRef,
	ldapGroundBindingName string,
) (*rbacv1.RoleBinding, error) {
	name := GenBindingName(username, roleRef.Name, ldapGroundBindingName)
	binding, err := svc.clientset.RbacV1().
		RoleBindings(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		svc.logger.Error("Failed to get role binding", "error", err)
		return nil, err
	}

	binding.RoleRef = roleRef
	binding, err = svc.clientset.RbacV1().
		RoleBindings(namespace).
		Update(ctx, binding, metav1.UpdateOptions{})
	if err != nil {
		svc.logger.Error("Failed to update role binding", "error", err)
		return nil, err
	}

	svc.logger.Info("Updated role binding", "name", binding.Name, "namespace", namespace)
	return binding, nil
}

// DeleteRoleBinding deletes a role binding by its name and namespace.
func (svc *serviceAccountsService) DeleteRoleBinding(
	ctx context.Context,
	namespace string,
	name string,
) error {
	err := svc.clientset.RbacV1().
		RoleBindings(namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		svc.logger.Error("Failed to delete role binding", "error", err)
		return err
	}

	svc.logger.Info("Deleted role binding", "name", name, "namespace", namespace)
	return nil
}

func (svc *serviceAccountsService) createServiceAccount(
	ctx context.Context,
	username string,
) (*corev1.ServiceAccount, error) {
	sa, err := svc.clientset.CoreV1().
		ServiceAccounts(svc.namespace).
		Create(ctx, &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: username,
			},
		}, metav1.CreateOptions{})
	if err != nil {
		svc.logger.Error("Failed to create service account", "error", err)
		return nil, err
	}

	svc.logger.Info("Created new service account", "name", sa.Name, "namespace", sa.Namespace)
	return sa, nil
}

func GenBindingName(username string, roleName string, ldapGroundBindingName string) string {
	return fmt.Sprintf("kerbernetes:%s:%s:%s", username, ldapGroundBindingName, roleName)
}

// int64Ptr is a helper function to create a pointer to an int64 value.
func int64Ptr(i int64) *int64 {
	return &i
}
