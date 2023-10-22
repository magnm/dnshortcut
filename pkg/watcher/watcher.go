package watcher

import (
	"log/slog"
	"time"

	"github.com/magnm/dnshortcut/pkg/coredns"
	"github.com/magnm/dnshortcut/pkg/kubernetes"
	"github.com/magnm/dnshortcut/pkg/watches"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Watcher struct {
	Watches        []watches.Watched
	configFactory  informers.SharedInformerFactory
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
	kubeClient, err := kubernetes.GetKubernetesClient()
	if err != nil {
		panic(err)
	}
	dynamicClient, err := kubernetes.GetKubernetesDynamicClient()
	if err != nil {
		panic(err)
	}

	w.dynamicFactory = dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute*5)

	stop := make(chan struct{})
	defer close(stop)

	w.setupConfigWatcher(kubeClient)
	// Wait for configWatcher to be synced before starting resource informers
	w.configFactory.WaitForCacheSync(stop)

	w.setupIngressInformers()

	w.configFactory.Start(stop)
	w.dynamicFactory.Start(stop)
	for {
		time.Sleep(time.Second)
	}
}

func (w *Watcher) setupConfigWatcher(clientset *k8s.Clientset) {
	w.configFactory = informers.NewSharedInformerFactoryWithOptions(
		clientset,
		time.Minute*5,
		informers.WithNamespace("kube-system"),
		informers.WithTweakListOptions(
			func(lo *metav1.ListOptions) {
				lo.LabelSelector = "app=coredns"
			},
		),
	)
	informer := w.configFactory.Core().V1().ConfigMaps().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			slog.Info("configmap added", "obj", obj)
			configMap := obj.(*corev1.ConfigMap)
			if configMap.Name == coredns.CustomConfigMapName {
				slog.Info("coredns configmap added", "obj", obj)
				coredns.IngressHostFile = string(configMap.Data[coredns.CustomHostfileName])
				coredns.ScheduleReconcile()
			}
		},
		DeleteFunc: func(obj interface{}) {
			slog.Info("configmap deleted", "obj", obj)
			// Unhandled, this isn't really expected, and doesn't matter
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			slog.Info("configmap updated", "oldObj", oldObj, "newObj", newObj)
			configMap := newObj.(*corev1.ConfigMap)
			if configMap.Name == coredns.CustomConfigMapName {
				slog.Info("coredns configmap updated", "oldObj", oldObj, "newObj", newObj)
				coredns.IngressHostFile = string(configMap.Data[coredns.CustomHostfileName])
				coredns.ScheduleReconcile()
			}
		},
	})
}

func (w *Watcher) setupIngressInformers() {
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
					serviceIp := watched.GetServiceIp(obj)
					coredns.AddIngress(hostname, serviceIp)
				}
			},
			DeleteFunc: func(obj interface{}) {
				slog.Info("object deleted", "resource", watched.APIResource(), "obj", obj)
				hostname := watched.GetHostname(obj)
				if hostname != "" {
					slog.Info("hostname deleted", "resource", watched.APIResource(), "hostname", hostname)
					coredns.RemoveIngress(hostname)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				slog.Info("object updated", "resource", watched.APIResource(), "oldObj", oldObj, "newObj", newObj)
				oldHostname := watched.GetHostname(oldObj)
				newHostname := watched.GetHostname(newObj)
				if oldHostname != newHostname {
					slog.Info("hostname updated", "resource", watched.APIResource(), "oldHostname", oldHostname, "newHostname", newHostname)
					coredns.RemoveIngress(oldHostname)
					serviceIp := watched.GetServiceIp(newObj)
					coredns.AddIngress(newHostname, serviceIp)
				}
			},
		})
	}
}
