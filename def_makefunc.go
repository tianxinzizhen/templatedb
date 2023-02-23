package templatedb

import (
	"context"
	"database/sql"
	"reflect"
)

var (
	contextType       = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType         = reflect.TypeOf((*error)(nil)).Elem()
	ResultType        = reflect.TypeOf((*Result)(nil))
	PrepareResultType = reflect.TypeOf((*PrepareResult)(nil))
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
	AutoCommit func(errp *error)
	Recover    func(errp *error)
}

type Result struct {
	LastInsertId int64
	RowsAffected int64
}
