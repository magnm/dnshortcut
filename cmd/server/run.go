package server

import "github.com/magnm/dnshortcut/pkg/watcher"

func Run() {
	watcher := watcher.NewWatcher()
	watcher.Watch()
}
