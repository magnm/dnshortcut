package watches

import (
	"log/slog"

	projectcontour "github.com/projectcontour/contour/apis/projectcontour/v1"
)

type HTTPProxy struct {
}

func (w *HTTPProxy) APIGroup() string {
	return "projectcontour.io"
}

func (w *HTTPProxy) APIVersion() string {
	return "v1"
}

func (w *HTTPProxy) APIResource() string {
	return "httpproxies"
}

func (w *HTTPProxy) GetHostname(obj interface{}) string {
	proxy, ok := obj.(*projectcontour.HTTPProxy)
	if !ok {
		slog.Error("failed to convert object to httpproxy", "obj", obj)
		return ""
	}

	return proxy.Spec.VirtualHost.Fqdn
}

func (w *HTTPProxy) GetServiceIp(obj interface{}) string {
	_, ok := obj.(*projectcontour.HTTPProxy)
	if !ok {
		slog.Error("failed to convert object to httpproxy", "obj", obj)
		return ""
	}

	// TODO
	return ""
}
