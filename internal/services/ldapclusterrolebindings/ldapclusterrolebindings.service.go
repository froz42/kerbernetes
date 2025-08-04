package ldapclusterrolebindingssvc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	k8ssvc "github.com/froz42/kerbernetes/internal/services/k8s"
	v1 "github.com/froz42/kerbernetes/k8s/api/rbac.kerbernetes.io/v1"
	clientset "github.com/froz42/kerbernetes/k8s/generated/clientset/versioned/typed/rbac.kerbernetes.io/v1"
	"github.com/samber/do"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LdapClusterRoleBindingService interface {
	Start(ctx context.Context) error
	GetBindings() []v1.LdapClusterRoleBindingSpec
}

type ldapClusterRoleBindingService struct {
	logger    *slog.Logger
	clientSet *clientset.RbacKerbenetesV1Client

	cache   []*v1.LdapClusterRoleBinding
	cacheMu sync.RWMutex

	informer cache.SharedIndexInformer
	stopCh   chan struct{}

	defaultNamespace string
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

	svc := &ldapClusterRoleBindingService{
		logger:    logger.With("service", "ldapclusterrolebindings"),
		clientSet: cs,
		cache:     []*v1.LdapClusterRoleBinding{},
		stopCh:    make(chan struct{}),
		defaultNamespace: k8sSvc.GetNamespace(),
	}

	svc.initInformer()

	return svc, nil
}

// initInformer sets up the informer with handlers
func (svc *ldapClusterRoleBindingService) initInformer() {
	svc.informer = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return svc.clientSet.LdapClusterRoleBindings(svc.defaultNamespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return svc.clientSet.LdapClusterRoleBindings(svc.defaultNamespace).Watch(context.TODO(), options)
			},
		},
		&v1.LdapClusterRoleBinding{},
		0,
		cache.Indexers{},
	)

	svc.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			binding := obj.(*v1.LdapClusterRoleBinding)
			svc.logger.Info("LdapClusterRoleBinding added", "name", binding.Name)
			svc.add(binding)
		},
		UpdateFunc: func(_, newObj any) {
			binding := newObj.(*v1.LdapClusterRoleBinding)
			svc.logger.Info("LdapClusterRoleBinding updated", "name", binding.Name)
			svc.update(binding)
		},
		DeleteFunc: func(obj any) {
			binding := obj.(*v1.LdapClusterRoleBinding)
			svc.logger.Info("LdapClusterRoleBinding deleted", "name", binding.Name)
			svc.delete(binding)
		},
	})
}

func (svc *ldapClusterRoleBindingService) Start(ctx context.Context) error {
	svc.logger.Info("Starting LdapClusterRoleBinding informer")

	go svc.informer.Run(svc.stopCh)

	if !cache.WaitForCacheSync(svc.stopCh, svc.informer.HasSynced) {
		return fmt.Errorf("failed to sync informer cache")
	}

	svc.logger.Info("LdapClusterRoleBinding informer started and synced", "count", len(svc.cache))
	<-ctx.Done()
	close(svc.stopCh)
	return nil
}

// cache operations

func (svc *ldapClusterRoleBindingService) add(binding *v1.LdapClusterRoleBinding) {
	svc.cacheMu.Lock()
	defer svc.cacheMu.Unlock()
	svc.cache = append(svc.cache, binding)
	svc.logger.Info("LdapClusterRoleBinding added to cache", "name", binding.Name, "namespace", binding.Namespace)
}

func (svc *ldapClusterRoleBindingService) update(binding *v1.LdapClusterRoleBinding) {
	svc.cacheMu.Lock()
	defer svc.cacheMu.Unlock()
	for i, b := range svc.cache {
		if b.Name == binding.Name && b.Namespace == binding.Namespace {
			svc.cache[i] = binding
			return
		}
	}
	// if not found, add
	svc.cache = append(svc.cache, binding)
}

func (svc *ldapClusterRoleBindingService) delete(binding *v1.LdapClusterRoleBinding) {
	svc.cacheMu.Lock()
	defer svc.cacheMu.Unlock()
	out := svc.cache[:0]
	for _, b := range svc.cache {
		if b.Name != binding.Name || b.Namespace != binding.Namespace {
			out = append(out, b)
		}
	}
	svc.logger.Info("LdapClusterRoleBinding deleted from cache", "name", binding.Name, "namespace", binding.Namespace)
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
