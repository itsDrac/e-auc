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

	"github.com/itsDrac/e-auc/internal/cache"
	db "github.com/itsDrac/e-auc/internal/database"
	"github.com/itsDrac/e-auc/internal/service"
	"github.com/itsDrac/e-auc/internal/storage"

	"github.com/go-playground/validator/v10"
	"github.com/itsDrac/e-auc/pkg/utils"
	"github.com/jackc/pgx/v5"
	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	HTTPServer *http.Server
	Services   *service.Services
	conn       *pgx.Conn
	cache      *cache.RedisCache
}

var validate *validator.Validate

func New() *Server {
	// log := logger.NewLogger()
	validate = validator.New(validator.WithRequiredStructEnabled())
	host := utils.GetEnv("SERVER_HOST", "0.0.0.0")
	port := utils.GetEnv("SERVER_PORT", "8080")
	dbDsn := utils.GetEnv("DB_DSN", "")

	serverAddr := fmt.Sprintf("%s:%s", host, port)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, dbDsn)
	if err != nil {
		slog.Error("[DB] connection failed -> ", "error", err.Error())
		panic(err)
	}

	querier := db.New(conn)
	// userRepo := repository.NewUserrepo(querier)

	storage, err := storage.NewMinioStorage()
	if err != nil {
		slog.Error("[Storage] failed to initialize -> ", "error", err.Error())
		panic(err)
	}

	services, err := service.NewServices(querier, storage)
	if err != nil {
		slog.Error("[Service] failed to initialized -> ", "error", err.Error())
		panic(err)
	}

	cache, err := cache.NewRedisClient(ctx)
	if err != nil {
		slog.Error("[Cache] failed to initialized ->", "error", err.Error())
		panic(err)
	}

	if err := cache.Ping(ctx); err != nil {
		slog.Error("[Cache] Unable to ping ->", "error", err.Error())
	} else {
		slog.Info("[Cache] connected")
	}

	serv := &Server{
		Services: services,
		conn:     conn,
		cache:    cache,
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
	if err := s.cache.Close(); err != nil {
		slog.Error("[Cache] close failed ->", "error", err.Error())
		return err
	}

	// close db
	if err := s.conn.Close(shutCtx); err != nil {
		slog.Error("[DB] close failed -> ", "error", err.Error())
		return err
	}

	slog.Info("[SERVER] shutdown complete.")

	return nil
}
