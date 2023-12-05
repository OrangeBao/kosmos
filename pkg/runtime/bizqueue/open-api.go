package bizqueue

import (
	"fmt"
	"time"

	"k8s.io/client-go/util/workqueue"

	"github.com/kosmos.io/kosmos/pkg/runtime"
)

type rootK8sOpenApiSyncer struct {
}

func (r *rootK8sOpenApiSyncer) Reconcile(key string) error {
	fmt.Printf("rootK8sOpenApiSyncer:::   key %s \n", key)
	return nil
}

func NewRootK8sOpenApiSyncer() runtime.Reconciler {
	return &rootK8sOpenApiSyncer{}
}

func TestOpenApiController() {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	c := runtime.NewOpenApiController(queue, NewRootK8sOpenApiSyncer())

	stop := make(chan struct{})

	go func() {
		time.Sleep(3 * time.Second)
		queue.Add("hello~")
		time.Sleep(3 * time.Second)
		queue.Add("world~")
		time.Sleep(3 * time.Second)
		stop <- struct{}{}
	}()

	c.Run(1, stop)
}
