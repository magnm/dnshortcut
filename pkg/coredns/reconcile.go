package coredns

import (
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/magnm/dnshortcut/pkg/kubernetes"
)

const ReconcileDelay = 5 * time.Second

var (
	nextRun = time.Unix(0, 0)
	mutex   = &sync.Mutex{}
)

/*
 * Schedule a new reconcile, which will happen
 * at most once per ReconcileDelay and no later
 * than ReconcileDelay from time of call
 */
func ScheduleReconcile() {
	mutex.Lock()
	defer mutex.Unlock()

	if time.Now().After(nextRun) {
		nextRun = time.Now().Add(ReconcileDelay)
		go delayedReconcile()
	}
}

func AddIngress(hostname string, ip string) {
	hosts[hostname] = ip
	ScheduleReconcile()
}

func RemoveIngress(hostname string) {
	delete(hosts, hostname)
	ScheduleReconcile()
}

/**
 * Run reconciling after waiting for ReconcileDelay
 */
func delayedReconcile() {
	time.Sleep(ReconcileDelay)
	reconcile()
}

func reconcile() {
	// Read ingressHostFile,
	// and make sure it's consistent against
	// our known ingresses

	needsHostfileUpdate := false

	// Build a new IngressHostFile from hosts
	// If this file differs from the one we have stored from before,
	// that means it's time to update!
	newIngressHostFile := generateHostfile(hosts)
	if newIngressHostFile != IngressHostFile {
		IngressHostFile = newIngressHostFile
		needsHostfileUpdate = true
	}

	if needsHostfileUpdate {
		// Apply the new IngressHostFile
		data := map[string]string{
			CustomHostfileName: IngressHostFile,
		}
		applyCustomConfigMap(data)
	}
}

func generateHostfile(hosts map[string]string) string {
	newIngressHosts := strings.Builder{}
	newIngressHosts.WriteString("# Generated by dnshortcut\n")
	newIngressHosts.WriteString("hosts {")
	for hostname, ip := range hosts {
		newIngressHosts.WriteString("    ")
		newIngressHosts.WriteString(hostname)
		newIngressHosts.WriteString(" ")
		newIngressHosts.WriteString(ip)
		newIngressHosts.WriteString("\n")
	}
	newIngressHosts.WriteString("    fallthrough\n")
	newIngressHosts.WriteString("}")

	return strings.TrimSpace(newIngressHosts.String())
}

func applyCustomConfigMap(data map[string]string) {
	client, err := kubernetes.GetKubernetesClient()
	if err != nil {
		slog.Error("failed to get kubernetes client", "error", err)
	}

	kubernetes.ApplyConfigMap(client, CustomConfigMapName, CustomConfigMapNamespace, data)
}
