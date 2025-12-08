package db

import (
	"context"
	"database/sql"
	"errors"
)

func (d *DB) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	ctx = addTimeoutContext(ctx)
	return d.Db.ExecContext(ctx, query, args...)
}

func (d *DB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	ctx = addTimeoutContext(ctx)
	if d.Db == nil {
		d.log.Warn("sql.DB is nil")
		return nil, errors.New("[DB] underlying sql.DB is nil")
	}

	return d.Db.QueryContext(ctx, query, args...)
}

func (d *DB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	ctx = addTimeoutContext(ctx)
	if d.Db == nil {
		d.log.Warn("sql.DB is nil")
		return &sql.Row{}
	}

	return d.Db.QueryRowContext(ctx, query, args...)
}

func (d *DB) RunTx(ctx context.Context, fn func(ctx context.Context, tx *sql.Tx) error) error {
	ctx = addTimeoutContext(ctx)

	if d.Db == nil {
		d.log.Warn("sql.DB is nil")
		return errors.New("[DB] underlying sql.DB is nil")
	}

	tx, err := d.Db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(ctx, tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
