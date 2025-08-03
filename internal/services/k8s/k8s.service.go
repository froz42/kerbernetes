package k8ssvc

import (
	"context"
	"log/slog"
	"os"
	"time"

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
	AuthAccount(ctx context.Context, username string) (*k8smodels.Credentials, error)
}

type k8sService struct {
	apiConfig configsvc.Config
	clientset *kubernetes.Clientset
	logger    *slog.Logger
	namespace string
}

func NewProvider() func(i *do.Injector) (K8sService, error) {
	return func(i *do.Injector) (K8sService, error) {
		return New(
			do.MustInvoke[*slog.Logger](i),
			do.MustInvoke[configsvc.ConfigService](i).GetConfig(),
		)
	}
}

func New(
	logger *slog.Logger,
	apiConfig configsvc.Config,
) (K8sService, error) {
	namespace := apiConfig.Namespace
	namespaceBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err == nil {
		namespace = string(namespaceBytes)
		logger.Info("Using namespace from service account", "namespace", namespace)
	} else {
		logger.Warn("Failed to read namespace from service account, using configured namespace", "error", err, "namespace", namespace)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Warn("Failed to create in-cluster config, falling back to kubeconfig", "error", err)

		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
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
	return &k8sService{
		clientset: clientset,
		apiConfig: apiConfig,
		logger:    logger.With("service", "k8s"),
	}, nil
}

func (svc *k8sService) AuthAccount(ctx context.Context, username string) (*k8smodels.Credentials, error) {
	sa, err := svc.clientset.CoreV1().
		ServiceAccounts(svc.namespace).
		Get(ctx, username, metav1.GetOptions{})
	// if not found, create a new service account
	if err != nil {
		if errors.IsNotFound(err) {
			sa, err = svc.clientset.CoreV1().
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
		}
	} else {
		svc.logger.Info("Found existing service account", "name", sa.Name, "namespace", sa.Namespace)
	}
	// now we can issue a token for the service account
	tr, err := svc.clientset.CoreV1().ServiceAccounts(svc.namespace).
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
	svc.logger.Info("Issued token for service account", "name", sa.Name, "namespace", sa.Namespace)
	return &k8smodels.Credentials{
		Kind:       "ExecCredential",
		ApiVersion: "client.authentication.k8s.io/v1beta1",
		Status: &k8smodels.Status{
			Token:               tr.Status.Token,
			ExpirationTimestamp: tr.Status.ExpirationTimestamp.Format(time.RFC3339),
		},
	}, nil
}

func int64Ptr(i int64) *int64 {
	return &i
}
