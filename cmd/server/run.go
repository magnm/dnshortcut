package server

import "github.com/magnm/dnshortcut/pkg/watcher"

func Run() {
	watcher.Watch()
}
