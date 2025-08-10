package ldapgroupbindingssvc

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

type LdapGroupBindingService interface {
	Start(ctx context.Context) error
	GetBindings() []v1.LdapGroupBindingSpec
}

type ldapGroupBindingService struct {
	logger    *slog.Logger
	clientSet *clientset.Clientset

	cache   []*v1.LdapGroupBinding
	cacheMu sync.RWMutex

	informerFactory informers.SharedInformerFactory
	informer        lcrbinformer.LdapGroupBindingInformer
	stopCh          chan struct{}
}

func NewProvider() func(i *do.Injector) (LdapGroupBindingService, error) {
	return func(i *do.Injector) (LdapGroupBindingService, error) {
		return New(
			do.MustInvoke[*slog.Logger](i),
			do.MustInvoke[k8ssvc.K8sService](i),
		)
	}
}

func New(logger *slog.Logger, k8sSvc k8ssvc.K8sService) (LdapGroupBindingService, error) {
	cs, err := clientset.NewForConfig(k8sSvc.GetRestConfig())
	if err != nil {
		return nil, err
	}

	informerFactory := informers.NewSharedInformerFactory(cs, 0)
	informer := informerFactory.RbacKerbenetes().V1()

	svc := &ldapGroupBindingService{
		logger:          logger.With("service", "ldapclusterrolebindings"),
		clientSet:       cs,
		informerFactory: informerFactory,
		informer:        informer.LdapGroupBindings(),
		cache:           []*v1.LdapGroupBinding{},
		stopCh:          make(chan struct{}),
	}

	err = svc.initInformer()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize informer: %w", err)
	}

	return svc, nil
}

// initInformer sets up the informer with handlers
func (svc *ldapGroupBindingService) initInformer() error {
	_, err := svc.informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			binding := obj.(*v1.LdapGroupBinding)
			svc.logger.Info("LdapClusterRoleBinding added", "name", binding.Name)
			svc.add(binding)
		},
		UpdateFunc: func(_, newObj interface{}) {
			binding := newObj.(*v1.LdapGroupBinding)
			svc.logger.Info("LdapClusterRoleBinding updated", "name", binding.Name)
			svc.update(binding)
		},
		DeleteFunc: func(obj interface{}) {
			binding := obj.(*v1.LdapGroupBinding)
			svc.logger.Info("LdapClusterRoleBinding deleted", "name", binding.Name)
			svc.delete(binding)
		},
	})
	return err
}

func (svc *ldapGroupBindingService) Start(ctx context.Context) error {
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

func (svc *ldapGroupBindingService) add(binding *v1.LdapGroupBinding) {
	svc.cacheMu.Lock()
	defer svc.cacheMu.Unlock()
	svc.cache = append(svc.cache, binding)
	svc.logger.Info(
		"LdapClusterRoleBinding added to cache",
		"name",
		binding.Name,
	)
}

func (svc *ldapGroupBindingService) update(binding *v1.LdapGroupBinding) {
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

func (svc *ldapGroupBindingService) delete(binding *v1.LdapGroupBinding) {
	svc.cacheMu.Lock()
	defer svc.cacheMu.Unlock()
	out := []*v1.LdapGroupBinding{}
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

func (svc *ldapGroupBindingService) GetBindings() []v1.LdapGroupBindingSpec {
	svc.cacheMu.RLock()
	defer svc.cacheMu.RUnlock()
	// return a copy
	out := make([]v1.LdapGroupBindingSpec, len(svc.cache))
	for i, binding := range svc.cache {
		out[i] = binding.Spec
	}
	svc.logger.Info("Returning cached LdapClusterRoleBindings", "count", len(out))
	return out
}
