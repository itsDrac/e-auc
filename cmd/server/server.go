package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/itsDrac/e-auc/internal/dependency"

	"github.com/itsDrac/e-auc/pkg/utils"
	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	HTTPServer   *http.Server
	Dependencies *dependency.Dependencies
}

func New() *Server {
	host := utils.GetEnv("SERVER_HOST", "0.0.0.0")
	port := utils.GetEnv("SERVER_PORT", "8080")
	dbDsn := utils.GetEnv("DB_DSN", "")

	serverAddr := fmt.Sprintf("%s:%s", host, port)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dependencies, err := dependency.NewDependencies(ctx, dbDsn)
	if err != nil {
		slog.Error("[Dependency] failed to initialize -> ", "error", err.Error())
		panic(err)
	}

	serv := &Server{
		Dependencies: dependencies,
	}

	// builds router
	mux := serv.routes()
	serv.HTTPServer = &http.Server{
		Addr:         serverAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return serv
}

func (s *Server) Run() error {
	// s.Logger.Infof("[SERVER] running at -> " + s.HTTPServer.Addr)
	slog.Info("[SERVER] running -> ", "address", s.HTTPServer.Addr)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Run Server in the background
	go func() {
		if err := s.HTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("[SERVER] failed to serve -> ", "error", err.Error())
		}
	}()

	// Listen for the interrupt signal
	<-ctx.Done()
	slog.Info("[SERVER] shutdown signal received")

	// create shutdown context with 30 - sec timeout
	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop http server
	if err := s.HTTPServer.Shutdown(shutCtx); err != nil {
		slog.Error("[SERVER] shutdown failed -> ", "error", err.Error())
		return err
	}

	// close cache
	if err := s.Dependencies.Cache.Close(); err != nil {
		slog.Error("[Cache] close failed ->", "error", err.Error())
		return err
	}

	// close db
	if err := s.Dependencies.Conn.Close(shutCtx); err != nil {
		slog.Error("[DB] close failed -> ", "error", err.Error())
		return err
	}

	slog.Info("[SERVER] shutdown complete.")
	return nil
}
