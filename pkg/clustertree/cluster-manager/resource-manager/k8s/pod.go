package k8s

import (
	"context"
	"fmt"

	// resourcemanager "github.com/kosmos.io/kosmos/pkg/clustertree/cluster-manager/resource-manager"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodResourceHandler struct {
	// resourcemanager.ResourceHandler
	A string
}

func (p *PodResourceHandler) Create(ctx context.Context, obj client.Object, opts any) error {
	o := opts.(client.CreateOption)
	fmt.Println("hhhhh", o)
	return nil
}

func (p *PodResourceHandler) Delete(ctx context.Context, obj client.Object, opts any) error {
	o := opts.(string)
	fmt.Println("hhhhh", p.A, o)
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
