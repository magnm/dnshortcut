package coredns

import (
	"log/slog"
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
	slog.Info("reconciling corefile", "hosts", hosts)
	// Read ingressHostFile,
	// and make sure it's consistent against
	// our known ingresses

	needsHostfileUpdate := false

	// Build a new IngressHostFile from hosts
	// If this file differs from the one we have stored from before,
	// that means it's time to update!
	newZonefiles := generateZonefiles(hosts)
	if newZonefiles != IngressHostFile {
		IngressHostFile = newZonefiles
		needsHostfileUpdate = true
	}

	if needsHostfileUpdate {
		// Apply the new IngressHostFile
		data := map[string]string{
			CustomHostfileName: IngressHostFile,
		}
		applyCustomConfigMap(data)
	} else {
		slog.Info("corefile is up to date")
	}
}

func applyCustomConfigMap(data map[string]string) {
	slog.Info("applying custom configmap", "data", data)
	client, err := kubernetes.GetKubernetesClient()
	if err != nil {
		slog.Error("failed to get kubernetes client", "err", err)
	}

	err = kubernetes.ApplyConfigMap(client, CustomConfigMapName, CustomConfigMapNamespace, data)
	if err != nil {
		slog.Error("failed to apply custom configmap", "err", err)
	}
}
