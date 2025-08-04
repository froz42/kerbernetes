package k8ssvc

import (
	"context"
	"log/slog"
	"os"

	configsvc "github.com/froz42/kerbernetes/internal/services/config"
	k8smodels "github.com/froz42/kerbernetes/internal/services/k8s/models"
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
}

type k8sService struct {
	apiConfig configsvc.Config
	clientset *kubernetes.Clientset
	logger    *slog.Logger
	namespace string
}

func NewProvider() func(i *do.Injector) (K8sService, error) {
	return func(i *do.Injector) (K8sService, error) {
		logger := do.MustInvoke[*slog.Logger](i)
		config := do.MustInvoke[configsvc.ConfigService](i).GetConfig()
		return New(logger, config)
	}
}

func New(logger *slog.Logger, apiConfig configsvc.Config) (K8sService, error) {
	namespace, err := getNamespace(apiConfig, logger)
	if err != nil {
		return nil, err
	}

	clientset, err := createK8sClient(logger)
	if err != nil {
		return nil, err
	}

	return &k8sService{
		clientset: clientset,
		apiConfig: apiConfig,
		logger:    logger.With("service", "k8s"),
		namespace: namespace,
	}, nil
}

// AuthAccount retrieves or creates a service account for the given username and issues a token for it.
func (svc *k8sService) AuthAccount(ctx context.Context, username string) (*k8smodels.Credentials, error) {
	_, err := svc.UpsertServiceAccount(ctx, username)
	if err != nil {
		return nil, err
	}

	token, err := svc.issueToken(ctx, username)
	if err != nil {
		return nil, err
	}

	return &k8smodels.Credentials{
		Kind:       "ExecCredential",
		ApiVersion: "client.authentication.k8s.io/v1beta1",
		Status: &k8smodels.Status{
			Token:               token.Status.Token,
			ExpirationTimestamp: token.Status.ExpirationTimestamp.Time,
		},
	}, nil
}

// getNamespace retrieves the namespace from the service account or uses the configured namespace.
func getNamespace(apiConfig configsvc.Config, logger *slog.Logger) (string, error) {
	namespace := apiConfig.Namespace
	namespaceBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err == nil {
		namespace = string(namespaceBytes)
		logger.Info("Using namespace from service account", "namespace", namespace)
	} else {
		logger.Warn("Failed to read namespace from service account, using configured namespace", "error", err, "namespace", namespace)
	}
	return namespace, nil
}

// createK8sClient creates a Kubernetes clientset using in-cluster configuration or kubeconfig.
func createK8sClient(logger *slog.Logger) (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Warn("Failed to create in-cluster config, falling back to kubeconfig", "error", err)

		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(
			loadingRules,
			nil,
			nil,
		)
		config, err = clientConfig.ClientConfig()
		if err != nil {
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
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

// issueToken creates a token for the service account.
func (svc *k8sService) issueToken(ctx context.Context, username string) (*authv1.TokenRequest, error) {
	token, err := svc.clientset.CoreV1().ServiceAccounts(svc.namespace).
		CreateToken(ctx, username, &authv1.TokenRequest{
			Spec: authv1.TokenRequestSpec{
				Audiences:         []string{"https://kubernetes.default.svc.cluster.local"},
				ExpirationSeconds: int64Ptr(int64(svc.apiConfig.TokenDuration)),
			},
		}, metav1.CreateOptions{})
	if err != nil {
		svc.logger.Error("Failed to create token for service account", "error", err)
		return nil, err
	}

	svc.logger.Info("Issued token for service account", "name", username, "namespace", svc.namespace)
	return token, nil
}

// int64Ptr is a helper function to create a pointer to an int64 value.
func int64Ptr(i int64) *int64 {
	return &i
}
