package resourcemanager

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceHandler interface {
	Create(ctx context.Context, obj client.Object, opts any) error
	Delete(ctx context.Context, obj client.Object, opts any) error
	Update(ctx context.Context, obj client.Object, opts any) error
	Get(ctx context.Context, obj client.Object, opts any) error
	List(ctx context.Context, objs client.ObjectList, opts any) error
}

type PodResourceHandler interface {
	ResourceHandler
}

type NodeResourceHandler interface {
	ResourceHandler
}
