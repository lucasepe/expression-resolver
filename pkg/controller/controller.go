package controller

import (
	"context"
	"fmt"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	expressionv1alpha1 "github.com/lucasepe/expression-resolver/pkg/apis/expression/v1alpha1"
	clientset "github.com/lucasepe/expression-resolver/pkg/generated/clientset/versioned"
)

// Controller is the controller implementation for Expression resources
type Controller struct {
	// expressionClientset is a clientset for our own API group
	client clientset.Interface

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	queue workqueue.RateLimitingInterface

	informer cache.SharedInformer

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	//recorder record.EventRecorder
}

// New creates a new Controller.
func New(
	client clientset.Interface,
	queue workqueue.RateLimitingInterface,
	informer cache.SharedInformer) *Controller {
	ctrl := &Controller{
		client:   client,
		informer: informer,
		queue:    queue,
	}

	// add the event handlers - that just simply enqueue items
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctrl.addExpression,
		UpdateFunc: ctrl.updateExpression,
	})

	return ctrl
}

func (c *Controller) addExpression(obj interface{}) {
	echo, ok := obj.(*expressionv1alpha1.Expression)
	if !ok {
		klog.Errorf("unexpected object %v", obj)
		return
	}
	c.queue.Add(event{
		eventType: addExpression,
		newObj:    echo.DeepCopy(),
	})
}

func (c *Controller) updateExpression(oldObj, newObj interface{}) {
	klog.Info("updating expression")
	oldExp, ok := oldObj.(*expressionv1alpha1.Expression)
	if !ok {
		klog.Errorf("unexpected new object %v", newObj)
		return
	}
	exp, ok := newObj.(*expressionv1alpha1.Expression)
	if !ok {
		klog.Errorf("unexpected new object %v", newObj)
		return
	}
	c.queue.Add(event{
		eventType: updateExpression,
		oldObj:    oldExp.DeepCopy(),
		newObj:    exp.DeepCopy(),
	})
}

// Run begins watching and syncing.
func (c *Controller) Run(ctx context.Context, numWorkers int) error {
	// eventually catches a crash and logs an error
	defer utilruntime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	klog.Info("Starting Expression controller")

	go c.informer.Run(ctx.Done())

	// Wait for all involved caches to be synced, before
	// processing items from the queue is started
	klog.Info("waiting for informer caches to sync")
	if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
		err := fmt.Errorf("failed to wait for informers caches to sync")
		utilruntime.HandleError(err)
		return err
	}

	klog.Infof("starting %d workers", numWorkers)
	for i := 0; i < numWorkers; i++ {
		go wait.Until(func() {
			c.runWorker(ctx)
		}, time.Second, ctx.Done())
	}
	klog.Info("controller ready")

	<-ctx.Done()
	klog.Info("Stopping Expression controller")

	return nil
}
