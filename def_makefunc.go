package templatedb

import (
	"context"
	"database/sql"
	"reflect"
)

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
	ResultType  = reflect.TypeOf((*Result)(nil))
)

type Operation int

const (
	ExecAction Operation = iota
	PrepareAction
	SelectAction
	SelectScanAction
	ExecNoResultAction
)

type DBFunc[T any] struct {
	Begin      func() (*T, error)
	BeginTx    func(ctx context.Context, opts *sql.TxOptions) (*T, error)
	AutoCommit func(ctx context.Context, errp *error)
}

type Result struct {
	LastInsertId int64
	RowsAffected int64
}
type TXContext struct{}

type TXDeepContext struct{}

var BeginContextKey TXContext

var AutoCommitContextKey TXDeepContext

type AutoCommitContextDeep struct {
	Deep int
}

func (db DBFunc[T]) BeginContext(ctx context.Context) (context.Context, *T, error) {
	tx, ok := ctx.Value(&BeginContextKey).(*T)
	if !ok {
		var err error
		tx, err = db.Begin()
		if err != nil {
			return nil, nil, err
		}
		ctx = context.WithValue(ctx, &BeginContextKey, tx)
		ctx = context.WithValue(ctx, &AutoCommitContextKey, &AutoCommitContextDeep{})
	}
	if d, ok := ctx.Value(&AutoCommitContextKey).(*AutoCommitContextDeep); ok {
		d.Deep++
	}
	return ctx, tx, nil
}

func (db DBFunc[T]) BeginTxContext(ctx context.Context, opts *sql.TxOptions) (context.Context, *T, error) {
	tx, ok := ctx.Value(&BeginContextKey).(*T)
	if !ok {
		var err error
		tx, err = db.BeginTx(ctx, opts)
		if err != nil {
			return nil, nil, err
		}
		ctx = context.WithValue(ctx, &BeginContextKey, tx)
		ctx = context.WithValue(ctx, &AutoCommitContextKey, &AutoCommitContextDeep{})
	}
	if d, ok := ctx.Value(&AutoCommitContextKey).(*AutoCommitContextDeep); ok {
		d.Deep++
	}
	return ctx, tx, nil
}

func (db DBFunc[T]) AutoCommitContext(ctx context.Context, errp *error) {
	if *errp == nil {
		e := recover()
		if e != nil {
			switch err := e.(type) {
			case error:
				*errp = err
			default:
				panic(e)
			}
		}
		recoverPrintf(ctx, *errp)
	}
	d, ok := ctx.Value(&AutoCommitContextKey).(*AutoCommitContextDeep)
	if ok && d != nil {
		d.Deep--
		if d.Deep == 0 {
			db.AutoCommit(ctx, errp)
		}
	}
}

func (db DBFunc[T]) Recover(ctx context.Context, errp *error) {
	if *errp == nil {
		e := recover()
		if e != nil {
			switch err := e.(type) {
			case error:
				*errp = err
			default:
				panic(e)
			}
		}
		recoverPrintf(ctx, *errp)
	}
}
