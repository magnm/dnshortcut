package coredns

const CustomConfigMapName = "coredns-custom"
const CustomConfigMapNamespace = "kube-system"
const CustomHostfileName = "hosts.override"

var Corefile string
var IngressHostFile string

var hosts map[string]string
