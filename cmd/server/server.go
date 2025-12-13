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

	db "github.com/itsDrac/e-auc/internal/database"
	"github.com/itsDrac/e-auc/internal/service"

	"github.com/itsDrac/e-auc/pkg/utils"
	"github.com/jackc/pgx/v5"
	_ "github.com/joho/godotenv/autoload"
	"github.com/go-playground/validator/v10"
)

type Server struct {
	HTTPServer *http.Server
	Services   *service.Services
	conn       *pgx.Conn
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

	services, err := service.NewServices(querier)
	if err != nil {
		slog.Error("[Service] failed to initialized -> ", "error", err.Error())
		panic(err)
	}

	serv := &Server{
		Services: services,
		conn:     conn,
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
	slog.Info("[SERVER] running at -> ", "address", s.HTTPServer.Addr)

	// Create context that listens for the interrupt signal
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
	fmt.Println("Shuting down server.")

	// create shutdown context with 30 - sec timeout
	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.conn.Close(shutCtx); err != nil {
		slog.Error("[DB] failed to close -> ", "error", err.Error())
		return err
	}

	// Trigger graceful shutdown
	if err := s.HTTPServer.Shutdown(shutCtx); err != nil {
		slog.Error("[SERVER] shutdown failed -> ", "error", err.Error())
		return err
	}

	return nil
}
