package k8ssvc

import (
	"log/slog"
	"os"

	envsvc "github.com/froz42/kerbernetes/internal/services/env"
	"github.com/samber/do"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sService interface {
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
	if err != nil {
		return nil, err
	}

	return &k8sService{
		clientset:  clientset,
		env:        apiConfig,
		logger:     logger.With("service", "k8s"),
		namespace:  namespace,
		restConfig: restConfig,
	}, nil
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
