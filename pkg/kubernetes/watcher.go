package kubernetes

import (
	"log/slog"
	"time"

	"github.com/magnm/dnshortcut/pkg/watches"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

type Watcher struct {
	Watches        []watches.Watched
	factory        informers.SharedInformerFactory
	dynamicFactory dynamicinformer.DynamicSharedInformerFactory
}

func NewWatcher() *Watcher {
	return &Watcher{
		Watches: []watches.Watched{
			&watches.HTTPProxy{},
		},
	}
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

	w.setupInformers()

	stop := make(chan struct{})
	defer close(stop)
	w.factory.Start(stop)
	w.dynamicFactory.Start(stop)
	for {
		time.Sleep(time.Second)
	}
}

func (w *Watcher) setupInformers() {
	for _, watched := range w.Watches {
		informer := w.dynamicFactory.ForResource(schema.GroupVersionResource{
			Group:    watched.APIGroup(),
			Version:  watched.APIVersion(),
			Resource: watched.APIResource(),
		}).Informer()

		informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				slog.Info("object added", "resource", watched.APIResource(), "obj", obj)
				hostname := watched.GetHostname(obj)
				if hostname != "" {
					slog.Info("hostname added", "resource", watched.APIResource(), "hostname", hostname)
				}
			},
			DeleteFunc: func(obj interface{}) {
				slog.Info("object deleted", "resource", watched.APIResource(), "obj", obj)
				hostname := watched.GetHostname(obj)
				if hostname != "" {
					slog.Info("hostname deleted", "resource", watched.APIResource(), "hostname", hostname)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				slog.Info("object updated", "resource", watched.APIResource(), "oldObj", oldObj, "newObj", newObj)
				oldHostname := watched.GetHostname(oldObj)
				newHostname := watched.GetHostname(newObj)
				if oldHostname != newHostname {
					slog.Info("hostname updated", "resource", watched.APIResource(), "oldHostname", oldHostname, "newHostname", newHostname)
				}
			},
		})
	}
}
