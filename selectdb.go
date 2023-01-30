package templatedb

import (
	"context"
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

func (sdb *SelectDB[T]) Select(params any, name ...any) []T {
	return sdb.selectCommon(context.Background(), sdb.sqldb, params, reflect.SliceOf(sdb.t), sdb.sliceLen, name).Interface().([]T)
}
func (sdb *SelectDB[T]) SelectContext(ctx context.Context, params any, name ...any) []T {
	return sdb.selectCommon(ctx, sdb.sqldb, params, reflect.SliceOf(sdb.t), sdb.sliceLen, name).Interface().([]T)
}

func (sdb *SelectDB[T]) SelectFirst(params any, name ...any) T {
	return sdb.selectCommon(context.Background(), sdb.sqldb, params, sdb.t, 0, name).Interface().(T)
}

func (sdb *SelectDB[T]) SelectFirstContext(ctx context.Context, params any, name ...any) T {
	return sdb.selectCommon(ctx, sdb.sqldb, params, sdb.t, 0, name).Interface().(T)
}
