package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	expressionv1alpha1 "github.com/lucasepe/expression-resolver/pkg/apis/expression/v1alpha1"
	"github.com/lucasepe/expression-resolver/pkg/controller"
	clientset "github.com/lucasepe/expression-resolver/pkg/generated/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	defaultKubeconfig := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	if len(defaultKubeconfig) == 0 {
		defaultKubeconfig = clientcmd.RecommendedHomeFile
	}

	workers := flag.Int("workers", 1, "number of workers")

	resyncIn := flag.Duration("resync", 30*time.Second, "resync period for the shared informer")

	kubeconfig := flag.String(clientcmd.RecommendedConfigPathFlag,
		defaultKubeconfig, "absolute path to the kubeconfig file")

	klog.InitFlags(nil)
	flag.Parse()

	var cfg *rest.Config
	var err error
	if len(*kubeconfig) > 0 {
		cfg, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		klog.Fatalf("error building config: %s", err.Error())
	}

	expressionClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("error building expression clientset: %s", err.Error())
	}

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	listWatcher := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			// list Pods in the specified namespace
			return expressionClient.ExampleV1alpha1().Expressions(metav1.NamespaceAll).List(context.Background(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return expressionClient.ExampleV1alpha1().Expressions(metav1.NamespaceAll).Watch(context.Background(), options)
		},
	}

	// create the shared informer and resync every `resyncIn` user defined value
	informer := cache.NewSharedInformer(listWatcher, &expressionv1alpha1.Expression{}, *resyncIn)

	ctrl := controller.New(expressionClient, queue, informer)

	ctx, cancel := signal.NotifyContext(context.Background(), []os.Signal{
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGKILL,
		syscall.SIGHUP,
		syscall.SIGQUIT,
	}...)
	defer cancel()

	err = ctrl.Run(ctx, *workers)
	if err != nil {
		klog.Fatal("error running controller ", err)
	}
}
