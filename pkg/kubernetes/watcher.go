package kubernetes

import (
	"log/slog"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

type Watcher struct {
	factory        informers.SharedInformerFactory
	dynamicFactory dynamicinformer.DynamicSharedInformerFactory
}

func NewWatcher() *Watcher {
	return &Watcher{}
}

func (w *Watcher) Watch() {
	clientset, err := GetKubernetesClient()
	if err != nil {
		panic(err)
	}
	dynamicClient, err := GetKubernetesDynamicClient()
	if err != nil {
		panic(err)
	}

	w.factory = informers.NewSharedInformerFactory(clientset, time.Second*30)
	w.dynamicFactory = dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Second*30)

	w.setupHTTPProxyInformer()

	stop := make(chan struct{})
	defer close(stop)
	w.factory.Start(stop)
	w.dynamicFactory.Start(stop)
	for {
		time.Sleep(time.Second)
	}
}

func (w *Watcher) setupHTTPProxyInformer() {
	informer := w.dynamicFactory.ForResource(schema.GroupVersionResource{
		Group:    "projectcontour.io",
		Version:  "v1",
		Resource: "httpproxies",
	}).Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			slog.Info("httpproxy added", "obj", obj)
		},
		DeleteFunc: func(obj interface{}) {
			slog.Info("httpproxy deleted", "obj", obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			slog.Info("httpproxy updated", "oldObj", oldObj, "newObj", newObj)
		},
	})
}
