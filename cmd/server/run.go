package server

import "github.com/magnm/dnshortcut/pkg/kubernetes"

func Run() {
	watcher := kubernetes.NewWatcher()
	watcher.Watch()
}
