package main

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/PaesslerAG/gval"
)

func main() {
	configLoader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	cfg, err := configLoader.ClientConfig()
	if err != nil {
		panic(err)
	}

	// the base API path "/api" (legacy resource)
	cfg.APIPath = "api"
	// the Pod group and version "/v1" (group name is empty for legacy resources)
	cfg.GroupVersion = &corev1.SchemeGroupVersion
	// specify the serializer
	cfg.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	rc, err := rest.RESTClientFor(cfg)
	if err != nil {
		panic(err.Error())
	}

	res, err := eval(rc, "x+y", "expression-example-data", "params")
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", res)
}

func eval(rc *rest.RESTClient, exp, dataRef, key string) (string, error) {
	cm := &corev1.ConfigMap{}

	err := rc.Get().
		Namespace(metav1.NamespaceDefault).
		Resource("configmaps").
		Name(dataRef).
		Do(context.TODO()).
		Into(cm)
	if err != nil {
		return err.Error(), err
	}

	var inp map[string]interface{}
	err = json.Unmarshal([]byte(cm.Data[key]), &inp)
	if err != nil {
		return err.Error(), err
	}

	val, err := gval.Evaluate("x+y", inp)
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
