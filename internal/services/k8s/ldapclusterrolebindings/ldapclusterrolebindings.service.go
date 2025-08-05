package ldapclusterrolebindingssvc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	k8ssvc "github.com/froz42/kerbernetes/internal/services/k8s"
	v1 "github.com/froz42/kerbernetes/k8s/api/rbac.kerbernetes.io/v1"
	clientset "github.com/froz42/kerbernetes/k8s/generated/clientset/versioned"
	informers "github.com/froz42/kerbernetes/k8s/generated/informers/externalversions"
	lcrbinformer "github.com/froz42/kerbernetes/k8s/generated/informers/externalversions/rbac.kerbernetes.io/v1"
	"github.com/samber/do"
	"k8s.io/client-go/tools/cache"
)

type LdapClusterRoleBindingService interface {
	Start(ctx context.Context) error
	GetBindings() []v1.LdapClusterRoleBindingSpec
}

type ldapClusterRoleBindingService struct {
	logger    *slog.Logger
	clientSet *clientset.Clientset

	cache   []*v1.LdapClusterRoleBinding
	cacheMu sync.RWMutex

	informerFactory informers.SharedInformerFactory
	informer        lcrbinformer.LdapClusterRoleBindingInformer
	stopCh          chan struct{}
}

func NewProvider() func(i *do.Injector) (LdapClusterRoleBindingService, error) {
	return func(i *do.Injector) (LdapClusterRoleBindingService, error) {
		return New(
			do.MustInvoke[*slog.Logger](i),
			do.MustInvoke[k8ssvc.K8sService](i),
		)
	}
}

func New(logger *slog.Logger, k8sSvc k8ssvc.K8sService) (LdapClusterRoleBindingService, error) {
	cs, err := clientset.NewForConfig(k8sSvc.GetRestConfig())
	if err != nil {
		return nil, err
	}

	informerFactory := informers.NewSharedInformerFactory(cs, 0)
	informer := informerFactory.RbacKerbenetes().V1()

	svc := &ldapClusterRoleBindingService{
		logger:          logger.With("service", "ldapclusterrolebindings"),
		clientSet:       cs,
		informerFactory: informerFactory,
		informer:        informer.LdapClusterRoleBindings(),
		cache:           []*v1.LdapClusterRoleBinding{},
		stopCh:          make(chan struct{}),
	}

	err = svc.initInformer()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize informer: %w", err)
	}

	return svc, nil
}

// initInformer sets up the informer with handlers
func (svc *ldapClusterRoleBindingService) initInformer() error {
	_, err := svc.informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			binding := obj.(*v1.LdapClusterRoleBinding)
			svc.logger.Info("LdapClusterRoleBinding added", "name", binding.Name)
			svc.add(binding)
		},
		UpdateFunc: func(_, newObj interface{}) {
			binding := newObj.(*v1.LdapClusterRoleBinding)
			svc.logger.Info("LdapClusterRoleBinding updated", "name", binding.Name)
			svc.update(binding)
		},
		DeleteFunc: func(obj interface{}) {
			binding := obj.(*v1.LdapClusterRoleBinding)
			svc.logger.Info("LdapClusterRoleBinding deleted", "name", binding.Name)
			svc.delete(binding)
		},
	})
	return err
}

func (svc *ldapClusterRoleBindingService) Start(ctx context.Context) error {
	svc.logger.Info("Starting LdapClusterRoleBinding informer")

	svc.informerFactory.Start(svc.stopCh)

	if !cache.WaitForCacheSync(svc.stopCh, svc.informer.Informer().HasSynced) {
		return fmt.Errorf("failed to sync informer cache")
	}

	svc.logger.Info("LdapClusterRoleBinding informer started and synced")
	<-ctx.Done()
	close(svc.stopCh)
	return nil
}

// cache operations

func (svc *ldapClusterRoleBindingService) add(binding *v1.LdapClusterRoleBinding) {
	svc.cacheMu.Lock()
	defer svc.cacheMu.Unlock()
	svc.cache = append(svc.cache, binding)
	svc.logger.Info(
		"LdapClusterRoleBinding added to cache",
		"name",
		binding.Name,
	)
}

func (svc *ldapClusterRoleBindingService) update(binding *v1.LdapClusterRoleBinding) {
	svc.cacheMu.Lock()
	defer svc.cacheMu.Unlock()
	for i, b := range svc.cache {
		if b.Name == binding.Name {
			svc.cache[i] = binding
			svc.logger.Info(
				"LdapClusterRoleBinding updated in cache",
				"name",
				binding.Name,
			)
			return
		}
	}
	svc.logger.Warn(
		"LdapClusterRoleBinding update called but not found in cache",
		"name",
		binding.Name,
	)
	// if not found, add
	svc.cache = append(svc.cache, binding)
}

func (svc *ldapClusterRoleBindingService) delete(binding *v1.LdapClusterRoleBinding) {
	svc.cacheMu.Lock()
	defer svc.cacheMu.Unlock()
	out := []*v1.LdapClusterRoleBinding{}
	for _, b := range svc.cache {
		if b.Name != binding.Name {
			out = append(out, b)
		}
	}
	svc.logger.Info(
		"LdapClusterRoleBinding deleted from cache",
		"name",
		binding.Name,
	)
	svc.cache = out
}

func (svc *ldapClusterRoleBindingService) GetBindings() []v1.LdapClusterRoleBindingSpec {
	svc.cacheMu.RLock()
	defer svc.cacheMu.RUnlock()
	// return a copy
	out := make([]v1.LdapClusterRoleBindingSpec, len(svc.cache))
	for i, binding := range svc.cache {
		out[i] = binding.Spec
	}
	svc.logger.Info("Returning cached LdapClusterRoleBindings", "count", len(out))
	return out
}
