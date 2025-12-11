package main

import (
	"log/slog"
	"os"

	"github.com/itsDrac/e-auc/cmd/server"
	"github.com/itsDrac/e-auc/pkg/utils"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found or error loading it", "error", err)
	}

	var handler slog.Handler

	// Configure structured logging with slog
	logOptions := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	}
	env := utils.GetEnv("GO_ENV", "developement")
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, logOptions)
	} else {
		handler = slog.NewTextHandler(os.Stdout, logOptions)
	}
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
