package coredns

import "strings"

const CustomConfigMapName = "coredns-custom"
const CustomConfigMapNamespace = "kube-system"
const CustomHostfileName = "shortcut-hosts.server"

var Corefile string
var IngressHostFile string

var hosts = map[string]string{}

func generateZonefiles(hosts map[string]string) string {
	// Group into common base domains
	baseDomains := map[string][]string{}
	for hostname := range hosts {
		baseDomain := strings.Join(strings.Split(hostname, ".")[1:], ".")
		baseDomains[baseDomain] = append(baseDomains[baseDomain], hostname)
	}

	newFile := strings.Builder{}

	for baseDomain, hostnames := range baseDomains {
		newFile.WriteString(baseDomain)
		newFile.WriteString(":53 {\n")
		newFile.WriteString("    errors\n")
		newFile.WriteString("    hosts {\n")
		for _, hostname := range hostnames {
			ip := hosts[hostname]
			newFile.WriteString("        ")
			newFile.WriteString(ip)
			newFile.WriteString(" ")
			newFile.WriteString(hostname)
			newFile.WriteString("\n")
		}
		newFile.WriteString("        fallthrough\n")
		newFile.WriteString("    }\n")
		newFile.WriteString("    forward . /etc/resolv.conf\n")
		newFile.WriteString("    cache 60\n")
		newFile.WriteString("    loop\n")
		newFile.WriteString("    reload\n")
		newFile.WriteString("    loadbalance\n")
		newFile.WriteString("}\n")
	}

	return strings.TrimSpace(newFile.String())
}
