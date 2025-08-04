package k8ssvc

import (
	"context"
	"log/slog"
	"os"

	envsvc "github.com/froz42/kerbernetes/internal/services/env"
	"github.com/samber/do"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sService interface {
	// UpsertServiceAccount retrieves or creates a service account for the given username
	UpsertServiceAccount(ctx context.Context, username string) (*corev1.ServiceAccount, error)

	// IssueToken creates a token for the service account
	IssueToken(ctx context.Context, username string) (*authv1.TokenRequest, error)

	// GetNamespace retrieves the namespace from the service account or uses the configured namespace
	GetNamespace() string

	// GetClientset returns the Kubernetes clientset
	GetClientset() *kubernetes.Clientset

	// GetRestConfig returns the REST configuration for the Kubernetes client
	GetRestConfig() *rest.Config
}

type k8sService struct {
	env        envsvc.Env
	clientset  *kubernetes.Clientset
	logger     *slog.Logger
	namespace  string
	restConfig *rest.Config
}

func NewProvider() func(i *do.Injector) (K8sService, error) {
	return func(i *do.Injector) (K8sService, error) {
		logger := do.MustInvoke[*slog.Logger](i)
		env := do.MustInvoke[envsvc.EnvSvc](i).GetEnv()
		return New(logger, env)
	}
}

func New(logger *slog.Logger, apiConfig envsvc.Env) (K8sService, error) {
	namespace, err := getNamespace(apiConfig, logger)
	if err != nil {
		return nil, err
	}

	restConfig, err := rest.InClusterConfig()
	if err != nil {
		logger.Warn("Failed to create in-cluster config, falling back to kubeconfig", "error", err)

		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(
			loadingRules,
			nil,
			nil,
		)
		restConfig, err = clientConfig.ClientConfig()
		if err != nil {
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(restConfig)

	return &k8sService{
		clientset: clientset,
		env:       apiConfig,
		logger:    logger.With("service", "k8s"),
		namespace: namespace,
		restConfig: restConfig,
	}, nil
}

// UpsertServiceAccount retrieves or creates a service account for the given username.
func (svc *k8sService) UpsertServiceAccount(ctx context.Context, username string) (*corev1.ServiceAccount, error) {
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
func (svc *k8sService) IssueToken(ctx context.Context, username string) (*authv1.TokenRequest, error) {
	token, err := svc.clientset.CoreV1().ServiceAccounts(svc.namespace).
		CreateToken(ctx, username, &authv1.TokenRequest{
			Spec: authv1.TokenRequestSpec{
				Audiences:         []string{"https://kubernetes.default.svc.cluster.local"},
				ExpirationSeconds: int64Ptr(int64(svc.env.TokenDuration)),
			},
		}, metav1.CreateOptions{})
	if err != nil {
		svc.logger.Error("Failed to create token for service account", "error", err)
		return nil, err
	}

	svc.logger.Info("Issued token for service account", "name", username, "namespace", svc.namespace)
	return token, nil
}

// GetNamespace retrieves the namespace from the service account or uses the configured namespace.
func (svc *k8sService) GetNamespace() string {
	return svc.namespace
}

// GetClientset returns the Kubernetes clientset.
func (svc *k8sService) GetClientset() *kubernetes.Clientset {
	return svc.clientset
}

// GetRestConfig returns the REST configuration for the Kubernetes client.
func (svc *k8sService) GetRestConfig() *rest.Config {
	return svc.restConfig
}

func (svc *k8sService) createServiceAccount(ctx context.Context, username string) (*corev1.ServiceAccount, error) {
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

// int64Ptr is a helper function to create a pointer to an int64 value.
func int64Ptr(i int64) *int64 {
	return &i
}

// getNamespace retrieves the namespace from the service account or uses the configured namespace.
func getNamespace(env envsvc.Env, logger *slog.Logger) (string, error) {
	namespace := env.Namespace
	namespaceBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err == nil {
		namespace = string(namespaceBytes)
		logger.Info("Using namespace from service account", "namespace", namespace)
	} else {
		logger.Warn("Failed to read namespace from service account, using configured namespace", "error", err, "namespace", namespace)
	}
	return namespace, nil
}
