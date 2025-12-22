package main

import (
	"log/slog"
	"os"

	"github.com/itsDrac/e-auc/cmd/server"
	_ "github.com/itsDrac/e-auc/docs"
	"github.com/joho/godotenv"
)

// @title			E-Auction API
// @version		0.0.1
// @description	This is a sample server for an e-auction platform.
// @termsOfService	http://e-auction.local/terms/
// @contact.name	API Support
// @contact.url	http://www.e-auction.local/support
// @contact.email	support.query@e-auc.fun
// @license.name	MIT
// @license.url	https://opensource.org/licenses/MIT
// @host			localhost:8080
// @BasePath		/api/v1
func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found or error loading it", "error", err)
	}

	// Configure structured logging with slog
	logOptions := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
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
