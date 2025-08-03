package k8ssvc

import (
	"context"
	"log"
	"time"

	configsvc "github.com/froz42/kerbernetes/internal/services/config"
	k8smodels "github.com/froz42/kerbernetes/internal/services/k8s/models"
	"github.com/samber/do"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sService interface {
	AuthAccount(ctx context.Context, username string) (*k8smodels.Credentials, error)
}

type k8sService struct {
	apiConfig configsvc.Config
	clientset *kubernetes.Clientset
}

func NewProvider() func(i *do.Injector) (K8sService, error) {
	return func(i *do.Injector) (K8sService, error) {
		return New(
			do.MustInvoke[configsvc.ConfigService](i).GetConfig(),
		)
	}
}

func New(
	apiConfig configsvc.Config,
) (K8sService, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(
		loadingRules,
		nil,
		nil,
	)
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &k8sService{
		clientset: clientset,
		apiConfig: apiConfig,
	}, nil
}

func (svc *k8sService) AuthAccount(ctx context.Context, username string) (*k8smodels.Credentials, error) {
	sa, err := svc.clientset.CoreV1().
		ServiceAccounts("default").
		Get(ctx, username, metav1.GetOptions{})
	// if not found, create a new service account
	if err != nil {
		if errors.IsNotFound(err) {
			sa, err = svc.clientset.CoreV1().
				ServiceAccounts("default").
				Create(ctx, &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name: username,
					},
				}, metav1.CreateOptions{})
			if err != nil {
				return nil, err
			}
			log.Printf("Created service account: %s", sa.Name)
		}
	} else {
		log.Printf("Service account already exists: %s", sa.Name)
	}
	// now we can issue a token for the service account
	tr, err := svc.clientset.CoreV1().ServiceAccounts("default").
		CreateToken(ctx, username, &authv1.TokenRequest{
			Spec: authv1.TokenRequestSpec{
				Audiences:         []string{"https://kubernetes.default.svc.cluster.local"},
				ExpirationSeconds: int64Ptr(int64(svc.apiConfig.TokenDuration)),
			},
		}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	log.Printf("Issued token for service account: %s", sa.Name)
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
