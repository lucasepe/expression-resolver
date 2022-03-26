package controller

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/PaesslerAG/gval"
	expressionv1alpha1 "github.com/lucasepe/expression-resolver/pkg/apis/expression/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"
)

const maxRetries = 3

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextItem(ctx) {
	}
}

func (c *Controller) processNextItem(ctx context.Context) bool {
	obj, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(obj)

	err := c.processEvent(ctx, obj)
	if err == nil {
		c.queue.Forget(obj)
	} else if c.queue.NumRequeues(obj) < maxRetries {
		klog.Errorf("error processing event: %v, retrying", err)
		c.queue.AddRateLimited(obj)
	} else {
		klog.Errorf("error processing event: %v, max retries reached", err)
		c.queue.Forget(obj)
		utilruntime.HandleError(err)
	}

	return true
}

func (c *Controller) processEvent(ctx context.Context, obj interface{}) error {
	event, ok := obj.(event)
	if !ok {
		klog.Error("unexpected event ", obj)
		return nil
	}
	switch event.eventType {
	case addExpression:
		return c.processAddExpression(ctx, event.newObj.(*expressionv1alpha1.Expression))
	case updateExpression:
		return c.processUpdateExpression(
			ctx,
			event.oldObj.(*expressionv1alpha1.Expression),
			event.newObj.(*expressionv1alpha1.Expression),
		)
	}
	return nil
}

func (c *Controller) processAddExpression(ctx context.Context, exp *expressionv1alpha1.Expression) error {
	res, err := evalExpression(exp)
	if err != nil {
		return err
	}
	exp.Status.Result = res

	_, err = c.client.ExampleV1alpha1().
		Expressions(exp.GetNamespace()).
		UpdateStatus(ctx, exp, metav1.UpdateOptions{})
	return err
}

func (c *Controller) processUpdateExpression(
	ctx context.Context,
	oldExp, newExp *expressionv1alpha1.Expression,
) error {
	if !oldExp.HasChanged(newExp) {
		klog.Infof("expression '%s' has not changed, skipping", oldExp.GetName())
		return nil
	}

	res, err := evalExpression(newExp)
	if err != nil {
		return err
	}
	newExp.Status.Result = res

	_, err = c.client.ExampleV1alpha1().
		Expressions(newExp.GetNamespace()).
		UpdateStatus(ctx, newExp, metav1.UpdateOptions{})
	return err
}

func evalExpression(src *expressionv1alpha1.Expression) (string, error) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(src.Spec.Data), &data)
	if err != nil {
		return err.Error(), err
	}

	val, err := gval.Evaluate(src.Spec.Body, data)
	if err != nil {
		return err.Error(), err
	}

	return strval(val), nil
}

func strval(v interface{}) string {
	switch v := v.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case error:
		return v.Error()
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
