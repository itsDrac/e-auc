package db

import (
	"context"
	"log/slog"
	"time"

	"database/sql"

	// "github.com/itsDrac/e-auc/pkg/logger"
	_ "github.com/lib/pq"
)

type DB struct {
	// log    *logger.Logger
	Db     *sql.DB
	closed bool
}

func NewDB(ctx context.Context, dsn string) (*DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// connection pooling
	db.SetMaxOpenConns(25)                 // max total open connections
	db.SetMaxIdleConns(25)                 // max idle connections
	db.SetConnMaxLifetime(5 * time.Second) // recycle connections periodically

	pingCx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Test connection
	if err := db.PingContext(pingCx); err != nil {
		return nil, err
	}

	slog.Info("[DB] connection established...")

	return &DB{
		// log:    log,
		Db:     db,
		closed: false,
	}, nil

}

func (d *DB) Close(ctx context.Context) error {

	if d.closed {
		return nil
	}
	d.closed = true

	done := make(chan error, 1)

	go func() {
		done <- d.Db.Close()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
