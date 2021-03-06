package templaterouter

import (
	"time"

	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	kcoreclient "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
)

// ServiceLookup is an interface for fetching the service associated with the given endpoints
type ServiceLookup interface {
	LookupService(*api.Endpoints) (*api.Service, error)
}

func NewListWatchServiceLookup(svcGetter kcoreclient.ServicesGetter, resync time.Duration, namespace string) ServiceLookup {
	svcStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return svcGetter.Services(namespace).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return svcGetter.Services(namespace).Watch(options)
		},
	}
	go cache.NewReflector(lw, &api.Service{}, svcStore, resync).Run(wait.NeverStop)

	return &serviceLWLookup{
		store: svcStore,
	}
}

type serviceLWLookup struct {
	store cache.Store
}

func (c *serviceLWLookup) LookupService(endpoints *api.Endpoints) (*api.Service, error) {
	var rawSvc interface{}
	var ok bool
	var err error

	if rawSvc, ok, err = c.store.Get(endpoints); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.NewNotFound(schema.GroupResource{
			Group:    api.GroupName,
			Resource: "Service",
		}, endpoints.Name)
	}

	return rawSvc.(*api.Service), nil
}
