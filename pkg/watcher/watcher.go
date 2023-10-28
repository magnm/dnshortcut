package watcher

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/magnm/dnshortcut/pkg/coredns"
	"github.com/magnm/dnshortcut/pkg/kubernetes"
	"github.com/magnm/dnshortcut/pkg/watches"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Watcher struct {
	Watches        []watches.Watched
	configFactory  informers.SharedInformerFactory
	serviceFactory informers.SharedInformerFactory
	dynamicFactory dynamicinformer.DynamicSharedInformerFactory
}

func Watch() {
	kubeClient, err := kubernetes.GetKubernetesClient()
	if err != nil {
		panic(err)
	}
	dynamicClient, err := kubernetes.GetKubernetesDynamicClient()
	if err != nil {
		panic(err)
	}

	w := &Watcher{
		Watches: watches.Watches,
	}

	w.dynamicFactory = dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute*5)

	stop := make(chan struct{})
	defer close(stop)

	w.setupConfigWatcher(kubeClient)
	w.setupServiceWatcher(kubeClient)
	w.setupIngressInformers()

	wg := &sync.WaitGroup{}

	w.configFactory.Start(stop)
	w.serviceFactory.Start(stop)
	// Wait for configWatcher to be synced before starting dynamic resource informers
	w.configFactory.WaitForCacheSync(stop)
	w.serviceFactory.WaitForCacheSync(stop)

	w.dynamicFactory.Start(stop)

	wg.Add(1)
	slog.Info("watcher started")
	wg.Wait()
}

func (w *Watcher) setupConfigWatcher(clientset *k8s.Clientset) {
	w.configFactory = informers.NewSharedInformerFactoryWithOptions(
		clientset,
		time.Minute*5,
		informers.WithNamespace("kube-system"),
		informers.WithTweakListOptions(
			func(lo *metav1.ListOptions) {
				lo.FieldSelector = fmt.Sprintf("metadata.name=%s", coredns.CustomConfigMapName)
			},
		),
	)
	informer := w.configFactory.Core().V1().ConfigMaps().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
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
			configMap := newObj.(*corev1.ConfigMap)
			if configMap.Name == coredns.CustomConfigMapName {
				slog.Info("coredns configmap updated", "oldObj", oldObj, "newObj", newObj)
				coredns.IngressHostFile = string(configMap.Data[coredns.CustomHostfileName])
				coredns.ScheduleReconcile()
			}
		},
	})
}

func (w *Watcher) setupServiceWatcher(clientset *k8s.Clientset) {
	w.serviceFactory = informers.NewSharedInformerFactoryWithOptions(clientset, time.Minute*5)
	informer := w.serviceFactory.Core().V1().Services().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			service := obj.(*corev1.Service)
			slog.Info("service added", "name", service.GetName(), "namespace", service.GetNamespace())
			if service.Spec.Type == "LoadBalancer" && len(service.Status.LoadBalancer.Ingress) > 0 {
				externalIp := service.Status.LoadBalancer.Ingress[0].IP
				clusterIp := service.Spec.ClusterIP
				slog.Info("caching loadbalancer service",
					"name", service.GetName(),
					"namespace", service.GetNamespace(),
					"externalIp", externalIp,
					"clusterIp", clusterIp)
				watches.ServiceCache[externalIp] = clusterIp
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			service := newObj.(*corev1.Service)
			slog.Info("service updated", "name", service.GetName(), "namespace", service.GetNamespace())
			if service.Spec.Type == "LoadBalancer" && len(service.Status.LoadBalancer.Ingress) > 0 {
				externalIp := service.Status.LoadBalancer.Ingress[0].IP
				clusterIp := service.Spec.ClusterIP
				slog.Info("caching loadbalancer service", "externalIp", externalIp, "clusterIp", clusterIp)
				watches.ServiceCache[externalIp] = clusterIp
			}
		},
	})
}

func (w *Watcher) setupIngressInformers() {
	for _, watched := range w.Watches {
		slog.Info("setting up informer for", "resource", watched.APIResource())
		informer := w.dynamicFactory.ForResource(schema.GroupVersionResource{
			Group:    watched.APIGroup(),
			Version:  watched.APIVersion(),
			Resource: watched.APIResource(),
		}).Informer()

		informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				slog.Info("watched resource added", "resource", watched.APIResource(), "obj", obj)
				u := obj.(*unstructured.Unstructured)
				hostname := watched.GetHostname(u)
				if hostname != "" {
					serviceIp := watched.GetServiceIp(u)
					if serviceIp == "" {
						slog.Error("can't add hostname without ip", "resource", watched.APIResource(), "hostname", hostname)
						return
					}
					slog.Info("adding hostname", "resource", watched.APIResource(), "hostname", hostname, "ip", serviceIp)
					coredns.AddIngress(hostname, serviceIp)
				}
			},
			DeleteFunc: func(obj interface{}) {
				slog.Info("watched resource deleted", "resource", watched.APIResource(), "obj", obj)
				u := obj.(*unstructured.Unstructured)
				hostname := watched.GetHostname(u)
				if hostname != "" {
					slog.Info("removing hostname", "resource", watched.APIResource(), "hostname", hostname)
					coredns.RemoveIngress(hostname)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				slog.Info("watched resource updated", "resource", watched.APIResource(), "oldObj", oldObj, "newObj", newObj)
				oldU := oldObj.(*unstructured.Unstructured)
				newU := newObj.(*unstructured.Unstructured)
				oldHostname := watched.GetHostname(oldU)
				newHostname := watched.GetHostname(newU)
				if oldHostname != newHostname {
					slog.Info("removing hostname", "resource", watched.APIResource(), "oldHostname", oldHostname, "newHostname", newHostname)
					coredns.RemoveIngress(oldHostname)
					serviceIp := watched.GetServiceIp(newU)
					if serviceIp == "" {
						slog.Error("can't add hostname without ip", "resource", watched.APIResource(), "hostname", newHostname)
						return
					}
					slog.Info("addinghostname", "resource", watched.APIResource(), "hostname", newHostname, "ip", serviceIp)
					coredns.AddIngress(newHostname, serviceIp)
				}
			},
		})
	}
}
