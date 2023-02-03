package templatedb

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
)

type SelectDB[T any] struct {
	actionDB
	sqldb    sqlDB
	sliceLen int
	t        reflect.Type
}

func DBSelect[T any](db TemplateDB) *SelectDB[T] {
	if db, ok := db.(*DefaultDB); ok {
		return &SelectDB[T]{actionDB: db, sqldb: db.sqlDB, sliceLen: 10, t: reflect.TypeOf((*T)(nil)).Elem()}
	}
	if db, ok := db.(*TemplateTxDB); ok {
		return &SelectDB[T]{actionDB: db.actionDB, sqldb: db.tx, sliceLen: 10, t: reflect.TypeOf((*T)(nil)).Elem()}
	}
	return nil
}

func (sdb *SelectDB[T]) SliceLen(sliceLen int) *SelectDB[T] {
	sdb.sliceLen = sliceLen
	return sdb
}

func (sdb *SelectDB[T]) Select(params any, name ...any) T {
	return sdb.selectCommon(context.Background(), sdb.sqldb, params, sdb.t, sdb.sliceLen, name).Interface().(T)
}

func (sdb *SelectDB[T]) SelectContext(ctx context.Context, params any, name ...any) T {
	return sdb.selectCommon(ctx, sdb.sqldb, params, sdb.t, sdb.sliceLen, name).Interface().(T)
}

func DBConvertRows[T any](rows *sql.Rows) (T, error) {
	return DBConvertRowsCap[T](rows, 0)
}

func DBConvertRowsCap[T any](rows *sql.Rows, cap int) (T, error) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	columns, err := rows.ColumnTypes()
	if err != nil {
		return reflect.Zero(t).Interface().(T), err
	}
	var ret reflect.Value
	st := t
	if t.Kind() == reflect.Slice {
		if cap <= 0 {
			cap = 10
		}
		ret = reflect.MakeSlice(t, 0, cap)
		st = t.Elem()
	} else {
		ret = reflect.New(t).Elem()
	}
	dest := newScanDest(columns, st)
	for rows.Next() {
		receiver := newReceiver(st, columns, dest)
		err = rows.Scan(dest...)
		if err != nil {
			return reflect.Zero(t).Interface().(T), err
		}
		if t.Kind() == reflect.Slice {
			ret = reflect.Append(ret, receiver)
		} else {
			return receiver.Interface().(T), nil
		}
	}
	return ret.Interface().(T), nil
}

func DBConvertRow[T any](rows *sql.Rows) (T, error) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	columns, err := rows.ColumnTypes()
	if err != nil {
		return reflect.Zero(t).Interface().(T), err
	}
	if t.Kind() == reflect.Slice {
		return reflect.Zero(t).Interface().(T), fmt.Errorf("DBConvertRow not Convert Slice")
	}
	dest := newScanDest(columns, t)
	receiver := newReceiver(t, columns, dest)
	err = rows.Scan(dest...)
	if err != nil {
		return reflect.Zero(t).Interface().(T), err
	}
	return receiver.Interface().(T), nil
}
