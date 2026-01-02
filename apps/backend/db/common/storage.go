package common

import (
	"context"
	"database/sql"

	"github.com/stephenafamo/bob"
)

type Storage struct {
	db   bob.DB
	name string
}

type dbContextKey struct{}

var dbExecutorKey dbContextKey

func NewStorage(name string, db *sql.DB) *Storage {
	return &Storage{
		db:   bob.NewDB(db),
		name: name,
	}
}

// Executor returns either the transaction (if present in ctx) or the base DB.
func (s *Storage) Executor(ctx context.Context) bob.Executor {
	if exec, ok := ctx.Value(dbExecutorKey).(bob.Executor); ok && exec != nil {
		return exec
	}
	return s.db
}

// InTransaction runs fn in a DB transaction and injects the tx executor into the context.
func (s *Storage) InTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return s.db.RunInTx(ctx, nil, func(ctx context.Context, exec bob.Executor) error {
		ctx = context.WithValue(ctx, dbExecutorKey, exec)
		return fn(ctx)
	})
}
