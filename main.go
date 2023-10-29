package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/magnm/dnshortcut/cmd/server"
)

func main() {
	logLevel := slog.LevelInfo
	if level, _ := os.LookupEnv("LOG_LEVEL"); strings.ToLower(level) == "debug" {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})))

	server.Run()
}
