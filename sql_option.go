package tgsql

import (
	"context"
	"database/sql"
	"reflect"
)

type funcExecOption struct {
	ctx    context.Context
	param  any
	result []reflect.Value
	sql    string
	args   []any
	option int
	offset int
	db     any
	stmt   *sql.Stmt
}

func (op *funcExecOption) GetDB(ctx context.Context) any {
	if op.stmt != nil {
		return op.stmt
	}
	tx, ok := FromSqlTx(ctx)
	if ok && tx != nil {
		return tx
	}
	return op.db
}
