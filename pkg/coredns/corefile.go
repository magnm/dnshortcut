package coredns

const ConfigMapName = "coredns"

var Corefile string
var IngressHostFile string

var hosts map[string]string
