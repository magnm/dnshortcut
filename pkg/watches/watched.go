package watches

type Watched interface {
	APIGroup() string
	APIVersion() string
	APIResource() string

	GetHostname(obj interface{}) string
}
