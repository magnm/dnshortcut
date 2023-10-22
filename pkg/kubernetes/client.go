package kubernetes

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyv1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var clientset *kubernetes.Clientset
var dynamicClient *dynamic.DynamicClient

func getConfig() (*rest.Config, error) {
	if os.Getenv("KUBERNETES_SERVICE_HOST") == "" {
		home := homedir.HomeDir()
		kubeconfig := filepath.Join(home, ".kube", "config")

		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		return rest.InClusterConfig()
	}
}

func GetKubernetesClient() (*kubernetes.Clientset, error) {
	if clientset != nil {
		return clientset, nil
	}
	config, err := getConfig()
	if err != nil {
		slog.Error("failed to get kubernetes config", "error", err)
		return nil, err
	}

	clientset, err = kubernetes.NewForConfig(config)
	return clientset, err
}

func GetKubernetesDynamicClient() (*dynamic.DynamicClient, error) {
	if dynamicClient != nil {
		return dynamicClient, nil
	}
	config, err := getConfig()
	if err != nil {
		slog.Error("failed to get kubernetes config", "error", err)
		return nil, err
	}

	dynamicClient, err = dynamic.NewForConfig(config)
	return dynamicClient, err
}

func ApplyConfigMap(client *kubernetes.Clientset, name string, namespace string, data map[string]string) error {
	kind := "ConfigMap"
	apiVersion := "v1"
	_, err := client.CoreV1().ConfigMaps(namespace).Apply(context.Background(), &applyv1.ConfigMapApplyConfiguration{
		TypeMetaApplyConfiguration: applymetav1.TypeMetaApplyConfiguration{
			Kind:       &kind,
			APIVersion: &apiVersion,
		},
		ObjectMetaApplyConfiguration: &applymetav1.ObjectMetaApplyConfiguration{
			Name:      &name,
			Namespace: &namespace,
		},
		Data: data,
	}, metav1.ApplyOptions{
		FieldManager: "dnshortcut",
	})
	return err
}
