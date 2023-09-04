package templatedb

import (
	"context"
	"reflect"
)

type SqlTemplate[T any] struct {
	Ctx   context.Context
	Sql   string
	Param any
}

func (st SqlTemplate[T]) Query(tdb *DBFuncTemplateDB) (T, error) {
	op := &funcExecOption{}
	var result T
	op.result = append(op.result, reflect.ValueOf(result))
	op.ctx = st.Ctx
	op.sql = st.Sql
	op.param = st.Param
	var db sqlDB = tdb.db
	if op.ctx == nil {
		op.ctx = context.Background()
	} else {
		tx, ok := FromSqlTx(op.ctx)
		if ok && tx != nil {
			db = tx
		}
	}
	var err error
	op.sql, op.args, err = tdb.sqlTemplateBuild(op.ctx, op.sql, op.param)
	if err != nil {
		return result, err
	}
	err = tdb.query(db, op)
	if err != nil {
		return result, err
	}
	return op.result[0].Interface().(T), nil
}

func (st SqlTemplate[T]) Exec(tdb *DBFuncTemplateDB) (*Result, error) {
	op := &funcExecOption{}
	op.ctx = st.Ctx
	op.sql = st.Sql
	op.param = st.Param
	var db sqlDB = tdb.db
	if op.ctx == nil {
		op.ctx = context.Background()
	} else {
		tx, ok := FromSqlTx(op.ctx)
		if ok && tx != nil {
			db = tx
		}
	}
	var err error
	op.sql, op.args, err = tdb.sqlTemplateBuild(op.ctx, op.sql, op.param)
	if err != nil {
		return nil, err
	}
	result, err := tdb.exec(db, op)
	if err != nil {
		return nil, err
	}
	return result, nil
}
