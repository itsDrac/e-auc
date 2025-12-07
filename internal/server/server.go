package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/itsDrac/e-auc/internal/db"
	"github.com/itsDrac/e-auc/internal/repository"
	"github.com/itsDrac/e-auc/internal/service"
	"github.com/itsDrac/e-auc/pkg/logger"
	"github.com/itsDrac/e-auc/pkg/utils"
	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	HTTPServer  *http.Server
	UserService *service.UserService
	Logger      *logger.Logger
	Db          *db.DB
}

func New() *Server {
	mux := chi.NewMux()
	log := logger.NewLogger()
	host := utils.GetEnv("SERVER_HOST", "0.0.0.0")
	port := utils.GetEnv("SERVER_PORT", "8080")
	dbDsn := utils.GetEnv("DB_DSN", "")

	serverAddr := fmt.Sprintf("%s:%s", host, port)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := db.NewDB(ctx, dbDsn, log)
	if err != nil {
		log.Fatal("[DB] connection failed -> " + err.Error())
	}
	userRepo := repository.NewUserrepo(db)
	userService := service.NewUserService(userRepo)
	serv := &Server{
		Logger: log,
		HTTPServer: &http.Server{
			Addr:         serverAddr,
			Handler:      mux,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		UserService: userService,
		Db:          db,
	}

	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)

	serv.CommonRoutes(mux)
	serv.UserRoutes(mux)
	return serv
}

func (s *Server) Run() error {
	s.Logger.Infof("[SERVER] running at -> " + s.HTTPServer.Addr)
	// Create context that listens for the interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Run Server in the background
	go func() {
		if err := s.HTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.Logger.Fatal("[SERVER] failed to serve -> " + err.Error())
		}
	}()

	// Listen for the interrupt signal
	<-ctx.Done()

	// create shutdown context with 30 - sec timeout
	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.Db.Close(shutCtx); err != nil {
		s.Logger.Fatal("[DB] failed to close -> " + err.Error())
		return err
	}

	// Trigger graceful shutdown
	if err := s.HTTPServer.Shutdown(shutCtx); err != nil {
		s.Logger.Fatal("[SERVER] shutdown failed -> " + err.Error())
		return err
	}

	return nil
}
