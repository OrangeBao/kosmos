// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	kosmosv1alpha1 "github.com/kosmos.io/kosmos/pkg/apis/kosmos/v1alpha1"
	versioned "github.com/kosmos.io/kosmos/pkg/generated/clientset/versioned"
	internalinterfaces "github.com/kosmos.io/kosmos/pkg/generated/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/kosmos.io/kosmos/pkg/generated/listers/kosmos/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// VirtualClusterInformer provides access to a shared informer and lister for
// VirtualClusters.
type VirtualClusterInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.VirtualClusterLister
}

type virtualClusterInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewVirtualClusterInformer constructs a new informer for VirtualCluster type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewVirtualClusterInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredVirtualClusterInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredVirtualClusterInformer constructs a new informer for VirtualCluster type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredVirtualClusterInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.KosmosV1alpha1().VirtualClusters(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.KosmosV1alpha1().VirtualClusters(namespace).Watch(context.TODO(), options)
			},
		},
		&kosmosv1alpha1.VirtualCluster{},
		resyncPeriod,
		indexers,
	)
}

func (f *virtualClusterInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredVirtualClusterInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *virtualClusterInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&kosmosv1alpha1.VirtualCluster{}, f.defaultInformer)
}

func (f *virtualClusterInformer) Lister() v1alpha1.VirtualClusterLister {
	return v1alpha1.NewVirtualClusterLister(f.Informer().GetIndexer())
}
