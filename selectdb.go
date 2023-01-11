package templatedb

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/tianxinzizhen/templatedb/scaner"
	"github.com/tianxinzizhen/templatedb/template"
)

type AnyDB interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

type SelectDB[T any] struct {
	db       *DefaultDB
	selectdb AnyDB
}

func newScanDest(columns []*sql.ColumnType, t reflect.Type) []any {
	indexMap := make(map[int][]int, len(columns))
	for i, item := range columns {
		if t.Kind() == reflect.Struct {
			f, ok := template.GetFieldByName(t, item.Name())
			if ok {
				indexMap[i] = f.Index
			} else {
				panic(fmt.Sprintf("类型%v无法扫描字段：%s", t, item.Name()))
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
			panic("scan map key type not string")
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
		i := 0
		for ; i < t.NumOut(); i++ {
			destSlice = append(destSlice, &scaner.ParameterScaner{Column: columns[i]})
		}
		for ; i < len(columns); i++ {
			destSlice = append(destSlice, reflect.New(columns[i].ScanType()).Interface())
		}
		return destSlice
	} else {
		if len(columns) > 0 {
			destSlice = append(destSlice, &scaner.ParameterScaner{Column: columns[0]})
			for i := 1; i < len(columns); i++ {
				destSlice = append(destSlice, reflect.New(columns[i].ScanType()).Interface())
			}
		}
		return destSlice
	}
}

func DBSelect[T any](db any) *SelectDB[T] {
	if db, ok := db.(*DefaultDB); ok {
		return &SelectDB[T]{db: db, selectdb: db.sqlDB}
	}
	if db, ok := db.(*TemplateTxDB); ok {
		return &SelectDB[T]{db: db.db, selectdb: db.tx}
	}
	return nil
}

func (sdb *SelectDB[T]) newReceiver(columns []*sql.ColumnType, scanRows []any) (*T, []any) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if t.Kind() == reflect.Struct {
		dest := new(T)
		dv := reflect.ValueOf(dest).Elem()
		for _, v := range scanRows {
			if vi, ok := v.(*scaner.StructScaner); ok {
				vi.Dest = &dv
			}
		}
		return dest, scanRows
	} else if t.Kind() == reflect.Map && t.Key().Kind() == reflect.String {
		var ret *T = new(T)
		dest := reflect.MakeMapWithSize(reflect.MapOf(t.Key(), t.Elem()), len(columns))
		for _, v := range scanRows {
			if vi, ok := v.(*scaner.MapScaner); ok {
				vi.Dest = &dest
			}
		}
		*ret = dest.Interface().(T)
		return ret, scanRows
	} else if t.Kind() == reflect.Slice {
		var ret *T = new(T)
		dest := reflect.MakeSlice(reflect.SliceOf(t.Elem()), len(columns), len(columns))
		for _, v := range scanRows {
			if vi, ok := v.(*scaner.SliceScaner); ok {
				vi.Dest = &dest
			}
		}
		*ret = dest.Interface().(T)
		return ret, scanRows
	} else if t.Kind() == reflect.Func {
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
		var ret *T = new(T)
		*ret = dest.Interface().(T)
		return ret, scanRows
	} else {
		var ret *T = new(T)
		dest := reflect.ValueOf(ret).Elem()
		for _, v := range scanRows {
			if vi, ok := v.(*scaner.ParameterScaner); ok {
				vi.Dest = &dest
			}
		}
		return ret, scanRows
	}
}

func (sdb *SelectDB[T]) query(params any, name []any) (*sql.Rows, []*sql.ColumnType, error) {
	statement := getSkipFuncName(3, name)
	sql, args, err := sdb.db.templateBuild(statement, params)
	if err != nil {
		return nil, nil, err
	}
	rows, err := sdb.selectdb.Query(sql, args...)
	if err != nil {
		return nil, nil, err
	}
	columns, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, err
	}
	return rows, columns, nil
}

func (sdb *SelectDB[T]) Select(params any, name ...any) []*T {
	rows, columns, err := sdb.query(params, name)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	scanIndex := newScanDest(columns, reflect.TypeOf((*T)(nil)).Elem())
	ret := *(new([]*T))
	for rows.Next() {
		receiver, destSlice := sdb.newReceiver(columns, scanIndex)
		err = rows.Scan(destSlice...)
		if err != nil {
			panic(err)
		}
		ret = append(ret, receiver)
	}
	return ret
}

func (sdb *SelectDB[T]) SelectFirst(params any, name ...any) *T {
	rows, columns, err := sdb.query(params, name)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	scanIndex := newScanDest(columns, reflect.TypeOf((*T)(nil)))
	if rows.Next() {
		receiver, destSlice := sdb.newReceiver(columns, scanIndex)
		err = rows.Scan(destSlice...)
		if err != nil {
			panic(err)
		}
		return receiver
	} else {
		return nil
	}
}
