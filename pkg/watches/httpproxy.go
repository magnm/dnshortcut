package watches

import (
	"log/slog"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func init() {
	RegisterWatched(&HTTPProxy{})
}

type HTTPProxy struct{}

func (w *HTTPProxy) APIGroup() string {
	return "projectcontour.io"
}

func (w *HTTPProxy) APIVersion() string {
	return "v1"
}

func (w *HTTPProxy) APIResource() string {
	return "httpproxies"
}

func (w *HTTPProxy) GetHostname(obj *unstructured.Unstructured) string {
	fqdn, ok, err := unstructured.NestedString(obj.Object, "spec", "virtualhost", "fqdn")
	if !ok || err != nil {
		slog.Error("failed to find fqdn in httpproxy", "obj", obj)
		return ""
	}

	return fqdn
}

func (w *HTTPProxy) GetServiceIp(obj *unstructured.Unstructured) string {
	// Contour stores the external loadbalancer ip in the status if the HTTPProxy
	// resource, and that's all we have to find the Service clusterIp
	ingresses, ok, err := unstructured.NestedSlice(obj.Object, "status", "loadBalancer", "ingress")
	if !ok || err != nil || len(ingresses) == 0 {
		slog.Error("failed to find ingresses in httpproxy", "obj", obj, "ok", ok, "err", err)
		return ""
	}
	ingress, ok := ingresses[0].(map[string]any)
	if !ok {
		slog.Error("failed to convert ingress in httpproxy", "ingresses", ingresses)
		return ""
	}
	externalIp, ok, err := unstructured.NestedString(ingress, "ip")
	if !ok || err != nil {
		slog.Error("failed to find ip in httpproxy ingress", "obj", obj, "ok", ok, "err", err)
		return ""
	}

	// Lookup ingress ip in the service cache
	clusterIp, ok := ServiceCache[externalIp]
	if !ok {
		slog.Error("failed to find service in service cache", "ip", externalIp)
		return ""
	}
	slog.Info("found cluster ip for httpproxy ingress", "name", obj.GetName(), "ip", externalIp, "clusterIp", clusterIp)

	return clusterIp
}
