package db

import (
	"context"
	"time"
)

func addTimeoutContext(ctx context.Context) context.Context {
	c, _ := context.WithTimeout(ctx, 5*time.Second)
	return c
}
