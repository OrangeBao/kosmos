package openapi

import (
	"context"
	"fmt"

	// resourcemanager "github.com/kosmos.io/kosmos/pkg/clustertree/cluster-manager/resource-manager"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodResourceHandler struct {
	// resourcemanager.ResourceHandler
}

func (p *PodResourceHandler) Create(ctx context.Context, obj client.Object, opts any) error {
	o := opts.(client.CreateOption)
	fmt.Println("openapi", o)
	return nil
}

func (p *PodResourceHandler) Delete(ctx context.Context, obj client.Object, opts any) error {
	o := opts.(string)
	fmt.Println("openapi", o)
	return nil
}

func (p *PodResourceHandler) Update(ctx context.Context, obj client.Object, opts any) error {
	return nil
}

func (p *PodResourceHandler) Get(ctx context.Context, obj client.Object, opts any) error {
	return nil
}

func (p *PodResourceHandler) List(ctx context.Context, objs client.ObjectList, opts any) error {
	return nil
}
