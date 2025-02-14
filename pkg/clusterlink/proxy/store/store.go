package store

import (
	"context"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/client-go/dynamic"

	"github.com/kosmos.io/kosmos/pkg/utils"
)

// store implements storage.Interface like underlying storage,
// Providing Get/Watch/List resources from clusters.
type store struct {
	gvr       schema.GroupVersionResource
	namespace *utils.MultiNamespace
	// newClientFunc returns a resource client
	newClientFunc func() (dynamic.NamespaceableResourceInterface, error)

	versioner storage.Versioner
	prefix    string
}

var _ storage.Interface = &store{}

func newStore(gvr schema.GroupVersionResource, newClientFunc func() (dynamic.NamespaceableResourceInterface, error), multiNS *utils.MultiNamespace, versioner storage.Versioner, prefix string) *store {
	return &store{
		newClientFunc: newClientFunc,
		versioner:     versioner,
		prefix:        prefix,
		namespace:     multiNS,
		gvr:           gvr,
	}
}

func (s *store) Versioner() storage.Versioner {
	return s.versioner
}

// Get implements storage.Interface.
func (s *store) Get(ctx context.Context, key string, opts storage.GetOptions, objPtr runtime.Object) error {
	var namespace, name string
	// get name and namespace from path
	part1, part2 := s.splitKey(key)
	if part2 == "" {
		name = part1
	} else {
		namespace, name = part1, part2
	}
	if namespace != metav1.NamespaceAll && !s.namespace.Contains(namespace) {
		return apierrors.NewNotFound(s.gvr.GroupResource(), name)
	}

	client, err := s.client(namespace)
	if err != nil {
		return err
	}

	obj, err := client.Get(ctx, name, convertToMetaV1GetOptions(opts))
	if err != nil {
		return err
	}
	setEmptyManagedFields(obj)
	unstructuredObj := objPtr.(*unstructured.Unstructured)
	obj.DeepCopyInto(unstructuredObj)
	return nil
}

// GetList implements storage.Interface.
func (s *store) GetList(ctx context.Context, key string, opts storage.ListOptions, listObj runtime.Object) error {
	return s.List(ctx, key, opts, listObj)
}

// List implements storage.Interface.
func (s *store) List(ctx context.Context, key string, opts storage.ListOptions, listObj runtime.Object) error {
	// For cluster scope resources, key is /prefix. Parts are ["", ""]
	// For namespace scope resources, key is /prefix/namespace. Parts are [namespace, ""]
	namespace, _ := s.splitKey(key)

	reqNS, objFilter, shortCircuit := filterNS(s.namespace, namespace)
	if shortCircuit {
		return nil
	}

	client, err := s.client(reqNS)
	if err != nil {
		return err
	}

	objects, err := client.List(ctx, convertToMetaV1ListOptions(opts))
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	filteredItems := make([]unstructured.Unstructured, 0, len(objects.Items))
	for _, obj := range objects.Items {
		setEmptyManagedFields(obj)

		if objFilter != nil {
			obj := obj
			if objFilter(&obj) {
				filteredItems = append(filteredItems, obj)
			}
		}
	}
	if len(filteredItems) > 0 {
		objects.Items = filteredItems
	}

	objects.DeepCopyInto(listObj.(*unstructured.UnstructuredList))
	return nil
}

// WatchList implements storage.Interface.
func (s *store) WatchList(ctx context.Context, key string, opts storage.ListOptions) (watch.Interface, error) {
	return s.Watch(ctx, key, opts)
}

// Watch implements storage.Interface.
func (s *store) Watch(ctx context.Context, key string, opts storage.ListOptions) (watch.Interface, error) {
	// For cluster scope resources, key is /prefix. Parts are ["", ""]
	// For namespace scope resources, key is /prefix/namespace. Parts are [namespace, ""]
	namespace, _ := s.splitKey(key)
	reqNS, objFilter, shortCircuit := filterNS(s.namespace, namespace)
	if shortCircuit {
		return watch.NewEmptyWatch(), nil
	}

	client, err := s.client(reqNS)
	if err != nil {
		return nil, err
	}

	watcher, err := client.Watch(ctx, convertToMetaV1ListOptions(opts))
	if err != nil {
		return nil, err
	}
	watcher = watch.Filter(watcher, func(in watch.Event) (watch.Event, bool) {
		setEmptyManagedFields(in.Object)
		if objFilter != nil {
			return in, objFilter(in.Object)
		}
		return in, true
	})
	return watcher, nil
}

// Create implements storage.Interface.
func (s *store) Create(context.Context, string, runtime.Object, runtime.Object, uint64) error {
	return fmt.Errorf("create is not suppported in proxy store")
}

// Delete implements storage.Interface.
func (s *store) Delete(context.Context, string, runtime.Object, *storage.Preconditions, storage.ValidateObjectFunc, runtime.Object) error {
	return fmt.Errorf("delete is not suppported in proxy store")
}

// GuaranteedUpdate implements storage.Interface.
func (s *store) GuaranteedUpdate(_ context.Context, _ string, _ runtime.Object, _ bool, _ *storage.Preconditions, _ storage.UpdateFunc, _ runtime.Object) error {
	return fmt.Errorf("update is not suppported in proxy store")
}

// Count implements storage.Interface.
func (s *store) Count(string) (int64, error) {
	return 0, fmt.Errorf("count is not suppported in proxy store")
}

func (s *store) client(namespace string) (dynamic.ResourceInterface, error) {
	client, err := s.newClientFunc()
	if err != nil {
		return nil, err
	}

	if len(namespace) > 0 {
		return client.Namespace(namespace), nil
	}
	return client, nil
}

func (s *store) splitKey(key string) (string, string) {
	// a key is like:
	// - /prefix
	// - /prefix/name
	// - /prefix/namespace
	// - /prefix/namespace/name
	k := strings.TrimPrefix(key, s.prefix)
	k = strings.TrimPrefix(k, "/")
	parts := strings.SplitN(k, "/", 2)

	part0, part1 := parts[0], ""
	if len(parts) == 2 {
		part1 = parts[1]
	}
	return part0, part1
}

// filterNS returns the namespace to request and the filter function to filter objects.
// filter namespace, only watch cached namsspace resources
func filterNS(cached *utils.MultiNamespace, request string) (reqNS string, objFilter func(runtime.Object) bool, shortCircuit bool) {
	if cached.IsAll {
		reqNS = request
		return
	}

	if ns, ok := cached.Single(); ok {
		if request == metav1.NamespaceAll || request == ns {
			reqNS = ns
		} else {
			shortCircuit = true
		}
		return
	}

	if request == metav1.NamespaceAll {
		reqNS = metav1.NamespaceAll
		objFilter = objectFilter(cached)
	} else if cached.Contains(request) {
		reqNS = request
	} else {
		shortCircuit = true
	}
	return
}

func objectFilter(ns *utils.MultiNamespace) func(o runtime.Object) bool {
	return func(o runtime.Object) bool {
		accessor, err := meta.Accessor(o)
		if err != nil {
			return true
		}
		return ns.Contains(accessor.GetNamespace())
	}
}

// setEmptyManagedFields sets the managed fields to empty
func setEmptyManagedFields(obj interface{}) {
	if accssor, err := meta.Accessor(obj); err == nil {
		if accssor.GetManagedFields() != nil {
			accssor.SetManagedFields(nil)
		}
	}
}
