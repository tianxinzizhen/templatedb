package tgsql

import (
	"context"
	"database/sql"
	"reflect"
)

type SqlTemplate[T any] struct {
	Ctx   context.Context
	Sql   string
	Param any
}

func (st SqlTemplate[T]) Query(tdb *TgenSql) (T, error) {
	op := &funcExecOption{
		db: tdb.db,
	}
	var result T
	op.result = append(op.result, reflect.ValueOf(result))
	op.ctx = st.Ctx
	op.sql = st.Sql
	op.param = st.Param
	if op.ctx == nil {
		op.ctx = context.Background()
	} else {
		tx, ok := FromSqlTx(op.ctx)
		if ok && tx != nil {
			op.db = tx
		}
	}
	var err error
	op.sql, op.args, err = tdb.sqlTemplateBuild(op.ctx, op.sql, op.param)
	if err != nil {
		return result, err
	}
	err = tdb.query(op)
	if err != nil {
		return result, err
	}
	return op.result[0].Interface().(T), nil
}

func (st SqlTemplate[T]) Exec(tdb *TgenSql) (sql.Result, error) {
	op := &funcExecOption{
		db: tdb.db,
	}
	op.ctx = st.Ctx
	op.sql = st.Sql
	op.param = st.Param
	if op.ctx == nil {
		op.ctx = context.Background()
	} else {
		tx, ok := FromSqlTx(op.ctx)
		if ok && tx != nil {
			op.db = tx
		}
	}
	var err error
	op.sql, op.args, err = tdb.sqlTemplateBuild(op.ctx, op.sql, op.param)
	if err != nil {
		return nil, err
	}
	result, err := tdb.exec(op)
	if err != nil {
		return nil, err
	}
	return result, nil
}
