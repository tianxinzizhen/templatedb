package tgsql

import (
	"context"
	"database/sql"
	"reflect"
)

var (
	contextType   = reflect.TypeFor[context.Context]()
	errorType     = reflect.TypeFor[error]()
	sqlResultType = reflect.TypeFor[sql.Result]()
)

type Operation int

const (
	execAction Operation = iota
	prepareAction
	selectAction
	selectOneAction
	selectScanAction
	execNoResultAction
)

var MaxStackLen = 50

type sqlDB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type sqlPrepare interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

type sqlStmt interface {
	ExecContext(ctx context.Context, args ...any) (sql.Result, error)
}
