package templatedb

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/tianxinzizhen/templatedb/scaner"
	"github.com/tianxinzizhen/templatedb/template"
)

type SelectDB[T any] struct {
	actionDB
	sqldb sqlDB
}

func DBSelect[T any](db TemplateDB) *SelectDB[T] {
	if db, ok := db.(*DefaultDB); ok {
		return &SelectDB[T]{actionDB: db, sqldb: db.sqlDB}
	}
	if db, ok := db.(*TemplateTxDB); ok {
		return &SelectDB[T]{actionDB: db.actionDB, sqldb: db.tx}
	}
	return nil
}

var tempScanDest map[reflect.Type]any

// 创建临时扫描字段
func getTempScanDest(scanType reflect.Type) any {
	if tempScanDest == nil {
		tempScanDest = make(map[reflect.Type]any)
	}
	if dest, ok := tempScanDest[scanType]; !ok {
		dest := reflect.New(scanType).Interface()
		tempScanDest[scanType] = dest
		return dest
	} else {
		return dest
	}
}

func newScanDest(columns []*sql.ColumnType, t reflect.Type) []any {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	indexMap := make(map[int][]int, len(columns))
	for i, item := range columns {
		if t.Kind() == reflect.Struct {
			f, ok := template.GetFieldByName(t, item.Name())
			if ok {
				indexMap[i] = f.Index
			} else {
				panic(fmt.Errorf("类型%v无法扫描字段：%s", t, item.Name()))
			}
		}
	}
	destSlice := make([]any, 0, len(columns))
	if t.Kind() == reflect.Struct {
		for si := range columns {
			destSlice = append(destSlice, &scaner.StructScaner{Index: indexMap[si]})
		}
		return destSlice
	} else if t.Kind() == reflect.Map {
		if t.Key().Kind() != reflect.String {
			panic(fmt.Errorf("scan map key type not string"))
		}
		for _, v := range columns {
			destSlice = append(destSlice, &scaner.MapScaner{Column: v, Name: v.Name()})
		}
		return destSlice
	} else if t.Kind() == reflect.Slice {
		for i, v := range columns {
			destSlice = append(destSlice, &scaner.SliceScaner{Column: v, Index: i})
		}
		return destSlice
	} else if t.Kind() == reflect.Func {
		if t.NumIn() == 0 && t.NumOut() > 0 {
			i := 0
			for ; i < t.NumOut(); i++ {
				destSlice = append(destSlice, &scaner.ParameterScaner{Column: columns[i]})
			}
			for ; i < len(columns); i++ {
				destSlice = append(destSlice, getTempScanDest(columns[i].ScanType()))
			}
			return destSlice
		} else if t.NumOut() == 0 && t.NumIn() > 0 {
			i := 0
			for ; i < t.NumIn(); i++ {
				destSlice = append(destSlice, &scaner.ParameterScaner{Column: columns[i]})
			}
			for ; i < len(columns); i++ {
				destSlice = append(destSlice, getTempScanDest(columns[i].ScanType()))
			}
			return destSlice
		} else {
			panic(fmt.Errorf("scan func In(%d) Out(%d) not supported", t.NumIn(), t.NumOut()))
		}
	} else {
		if len(columns) > 0 {
			destSlice = append(destSlice, &scaner.ParameterScaner{Column: columns[0]})
			for i := 1; i < len(columns); i++ {
				destSlice = append(destSlice, getTempScanDest(columns[i].ScanType()))
			}
		}
		return destSlice
	}
}

func newReceiver(rt reflect.Type, columns []*sql.ColumnType, scanRows []any) reflect.Value {
	t := rt
	if rt.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	var ret reflect.Value = reflect.New(t)
	if t.Kind() == reflect.Struct {
		dv := ret.Elem()
		for _, v := range scanRows {
			if vi, ok := v.(*scaner.StructScaner); ok {
				vi.Dest = &dv
			}
		}
	} else if t.Kind() == reflect.Map && t.Key().Kind() == reflect.String {
		dest := reflect.MakeMapWithSize(reflect.MapOf(t.Key(), t.Elem()), len(columns))
		for _, v := range scanRows {
			if vi, ok := v.(*scaner.MapScaner); ok {
				vi.Dest = &dest
			}
		}
		ret.Elem().Set(dest)
	} else if t.Kind() == reflect.Slice {
		dest := reflect.MakeSlice(reflect.SliceOf(t.Elem()), len(columns), len(columns))
		for _, v := range scanRows {
			if vi, ok := v.(*scaner.SliceScaner); ok {
				vi.Dest = &dest
			}
		}
		ret.Elem().Set(dest)
	} else if t.Kind() == reflect.Func {
		if t.NumIn() == 0 && t.NumOut() > 0 {
			var results []reflect.Value = make([]reflect.Value, 0, t.NumOut())
			for i := 0; i < t.NumOut(); i++ {
				results = append(results, reflect.New(t.Out(i)).Elem())
			}
			dest := reflect.MakeFunc(t, func([]reflect.Value) []reflect.Value {
				return results
			})
			for i, v := range scanRows {
				if vi, ok := v.(*scaner.ParameterScaner); ok {
					vi.Dest = &results[i]
				}
			}
			ret.Elem().Set(dest)
		} else {
			var results []reflect.Value = make([]reflect.Value, 0, t.NumIn())
			for i := 0; i < t.NumIn(); i++ {
				results = append(results, reflect.New(t.In(i)).Elem())
			}
			for i, v := range scanRows {
				if vi, ok := v.(*scaner.ParameterScaner); ok {
					vi.Dest = &results[i]
				}
			}
			return reflect.ValueOf(results)
		}
	} else {
		dest := ret.Elem()
		for _, v := range scanRows {
			if vi, ok := v.(*scaner.ParameterScaner); ok {
				vi.Dest = &dest
			}
		}
	}
	if rt.Kind() == reflect.Pointer {
		return ret
	} else {
		return ret.Elem()
	}
}

func (sdb *SelectDB[T]) Select(params any, name ...any) []T {
	return sdb.selectContextCommon(context.Background(), params, name...)
}
func (sdb *SelectDB[T]) SelectContext(ctx context.Context, params any, name ...any) []T {
	return sdb.selectContextCommon(ctx, params, name...)
}

func (sdb *SelectDB[T]) selectContextCommon(ctx context.Context, params any, name ...any) []T {
	statement := getSkipFuncName(3, name)
	rows, columns, err := sdb.query(ctx, sdb.sqldb, statement, params, name)
	if err != nil {
		panic(fmt.Errorf("%s->%s", statement, err))
	}
	defer rows.Close()
	t := reflect.TypeOf((*T)(nil)).Elem()
	dest := newScanDest(columns, t)
	ret := *(new([]T))
	for rows.Next() {
		receiver := newReceiver(t, columns, dest)
		err = rows.Scan(dest...)
		if err != nil {
			panic(fmt.Errorf("%s->%s", statement, err))
		}
		ret = append(ret, receiver.Interface().(T))
	}
	return ret
}

func (sdb *SelectDB[T]) SelectFirst(params any, name ...any) T {
	return sdb.selectFirstContextCommon(context.Background(), params, name...)
}

func (sdb *SelectDB[T]) SelectFirstContext(ctx context.Context, params any, name ...any) T {
	return sdb.selectFirstContextCommon(ctx, params, name...)
}

func (sdb *SelectDB[T]) selectFirstContextCommon(ctx context.Context, params any, name ...any) (ret T) {
	statement := getSkipFuncName(3, name)
	rows, columns, err := sdb.query(ctx, sdb.sqldb, statement, params, name)
	if err != nil {
		panic(fmt.Errorf("%s->%s", statement, err))
	}
	defer rows.Close()
	t := reflect.TypeOf((*T)(nil)).Elem()
	dest := newScanDest(columns, t)
	if rows.Next() {
		receiver := newReceiver(t, columns, dest)
		err = rows.Scan(dest...)
		if err != nil {
			panic(fmt.Errorf("%s->%s", statement, err))
		}
		return receiver.Interface().(T)
	}
	return
}
