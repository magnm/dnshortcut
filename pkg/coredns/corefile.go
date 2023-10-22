package coredns

import "strings"

const CustomConfigMapName = "coredns-custom"
const CustomConfigMapNamespace = "kube-system"
const CustomHostfileName = "hosts.override"

var Corefile string
var IngressHostFile string

var hosts map[string]string

func generateHostfile(hosts map[string]string) string {
	newIngressHosts := strings.Builder{}
	newIngressHosts.WriteString("# Generated by dnshortcut\n")
	newIngressHosts.WriteString("hosts {")
	for hostname, ip := range hosts {
		newIngressHosts.WriteString("    ")
		newIngressHosts.WriteString(ip)
		newIngressHosts.WriteString(" ")
		newIngressHosts.WriteString(hostname)
		newIngressHosts.WriteString("\n")
	}
	newIngressHosts.WriteString("    fallthrough\n")
	newIngressHosts.WriteString("}")

	return strings.TrimSpace(newIngressHosts.String())
}
