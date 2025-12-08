package main

import (
	"log/slog"
	"os"

	"github.com/itsDrac/e-auc/cmd/server"
)

func main() {
	// Configure structured logging with slog
	logOptions := &slog.HandlerOptions{
		AddSource: true,
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(os.Stdout, logOptions)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Service initialization
	slog.Info("Initializing e-auction service...")
	
	server := server.New()
	if err := server.Run(); err != nil {
		slog.Error("server failed to run", "error", err)
		os.Exit(1)
	}
}
