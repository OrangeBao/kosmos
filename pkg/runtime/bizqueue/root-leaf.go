package bizqueue

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/workqueue"

	kosmosv1alpha1 "github.com/kosmos.io/kosmos/pkg/apis/kosmos/v1alpha1"
	"github.com/kosmos.io/kosmos/pkg/runtime"
	leafUtils "github.com/kosmos.io/kosmos/pkg/runtime/utils"
)

const (
	K8s     = "K8s"
	OpenApi = "OpenApi"
)

type rootK8sToLeafSyncer struct {
}

func (r *rootK8sToLeafSyncer) Reconcile(key string) error {
	fmt.Printf("rootK8sToLeafSyncer:::   key %s \n", key)
	// key -> node -> cluster
	globalleafManager := leafUtils.GetGlobalLeafResourceManager()

	lr, err := globalleafManager.GetLeafResource(key)
	if err != nil {
		// fmt.Errorf("err: %s", err)
		return err
	}

	return lr.Reconcile(key)
}

func NewRootK8sToLeafSyncer() runtime.Reconciler {
	return &rootK8sToLeafSyncer{}
}

func TestRootToLeaf() {
	globalleafManager := leafUtils.GetGlobalLeafResourceManager()

	clusterK8s := &kosmosv1alpha1.Cluster{}
	clusterK8s.Name = K8s
	clusterOpenApi := &kosmosv1alpha1.Cluster{}
	clusterOpenApi.Name = OpenApi
	globalleafManager.AddLeafResource(&leafUtils.K8sLeafResource{}, clusterK8s, []*corev1.Node{})
	globalleafManager.AddLeafResource(&leafUtils.OpenApiLeafResource{}, clusterOpenApi, []*corev1.Node{})

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	c := runtime.NewOpenApiController(queue, NewRootK8sToLeafSyncer())

	stop := make(chan struct{})

	go func() {
		time.Sleep(3 * time.Second)
		queue.Add(K8s)
		time.Sleep(3 * time.Second)
		queue.Add(OpenApi)
		time.Sleep(3 * time.Second)
		stop <- struct{}{}
	}()

	c.Run(1, stop)
}
