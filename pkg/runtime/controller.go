package runtime

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const (
	K8S int = iota
	OPENAPI
)

type Func func(key string) error

type Reconciler interface {
	Reconcile(key string) error
}

type Controller struct {
	// indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller
	kind     int
	Do       Reconciler
}

func NewOpenApiController(queue workqueue.RateLimitingInterface, r Reconciler) Controller {
	return Controller{
		queue: queue,
		kind:  OPENAPI,
		Do:    r,
	}
}

func NewK8sController(queue workqueue.RateLimitingInterface, informer cache.Controller, r Reconciler) Controller {
	return Controller{
		queue:    queue,
		kind:     K8S,
		informer: informer,
		Do:       r,
	}
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}
func (c *Controller) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key.
	defer c.queue.Done(key)

	// Invoke the method containing the business logic
	err := c.Reconcile(key.(string))
	// Handle the error if something went wrong during the execution of the business logic
	c.handleErr(err, key)
	return true
}

func (c *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	klog.Info("Starting Pod controller")

	if c.kind == K8S {
		go c.informer.Run(stopCh)

		// Wait for all involved caches to be synced, before processing items from the queue is started
		if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
			return
		}
	}

	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	klog.Info("Stopping Pod controller")
}

func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}

	klog.Infof("Reconcile error, key: %v, err: %s", key, err)

	c.queue.AddRateLimited(key)

	// // This controller retries 5 times if something goes wrong. After that, it stops trying.
	// if c.queue.NumRequeues(key) < 5 {
	// 	klog.Infof("Error syncing pod %v: %v", key, err)

	// 	// Re-enqueue the key rate limited. Based on the rate limiter on the
	// 	// queue and the re-enqueue history, the key will be processed later again.
	// 	c.queue.AddRateLimited(key)
	// 	return
	// }

	// c.queue.Forget(key)
	// // Report to an external entity that, even after several retries, we could not successfully process this key
	// runtime.HandleError(err)
	// klog.Infof("Dropping pod %q out of the queue: %v", key, err)
}

func (c *Controller) Reconcile(key string) error {
	if err := c.Do.Reconcile(key); err != nil {
		// TODO:
		fmt.Println(err.Error())
	}
	return nil
}
