package dependency

import (
	"context"
	"log/slog"

	"github.com/itsDrac/e-auc/internal/cache"
	db "github.com/itsDrac/e-auc/internal/database"
	"github.com/itsDrac/e-auc/internal/handlers"
	"github.com/itsDrac/e-auc/internal/service"
	"github.com/itsDrac/e-auc/internal/storage"
	"github.com/jackc/pgx/v5"
)

// Dependencies holds all the intialized instances required by the application.
type Dependencies struct {
	Services       *service.Services
	Conn           *pgx.Conn
	Cache          cache.Cacher
	UserHandler    *handlers.UserHandler
	ProductHandler *handlers.ProductHandler
}

// NewDependencies connects to DB, and wires up all services
func NewDependencies(ctx context.Context, dbDsn string) (*Dependencies, error) {

	conn, err := pgx.Connect(ctx, dbDsn)
	if err != nil {
		slog.Error("[DB] connection failed -> ", "error", err.Error())
		return nil, err
	}

	querier := db.New(conn)

	storage, err := storage.NewMinioStorage()
	if err != nil {
		slog.Error("[Storage] failed to initialize -> ", "error", err.Error())
		return nil, err
	}

	services, err := service.NewServices(querier, storage)
	if err != nil {
		slog.Error("[Service] failed to initialized -> ", "error", err.Error())
		return nil, err
	}

	cache, err := cache.NewRedisClient(ctx)
	if err != nil {
		slog.Error("[Cache] failed to initialized ->", "error", err.Error())
		return nil, err
	}

	if err := cache.Ping(ctx); err != nil {
		slog.Error("[Cache] Unable to ping ->", "error", err.Error())
	} else {
		slog.Info("[Cache] connected")
	}

	userHandler, err := handlers.NewUserHandler(services.UserService, services.AuthService, cache)
	if err != nil {
		slog.Error("[User Handler] failed to initialized -> ", "error", err.Error())
		return nil, err
	}

	productHandler, err := handlers.NewProductHandler(services.ProductService, cache)
	if err != nil {
		slog.Error("[Product Handler] failed to initialized -> ", "error", err.Error())
		return nil, err
	}

	return &Dependencies{
		Services:       services,
		Conn:           conn,
		Cache:          cache,
		ProductHandler: productHandler,
		UserHandler:    userHandler,
	}, nil

}
