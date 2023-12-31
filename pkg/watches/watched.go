package watches

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

type Watched interface {
	APIGroup() string
	APIVersion() string
	APIResource() string

	GetHostname(obj *unstructured.Unstructured) string
	GetServiceIp(obj *unstructured.Unstructured) string
}

var Watches = []Watched{}

var ServiceCache = map[string]string{}

func RegisterWatched(watched Watched) {
	Watches = append(Watches, watched)
}
