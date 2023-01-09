package templatedb

import (
	"database/sql"
	"reflect"

	"github.com/tianxinzizhen/templatedb/template"
	"github.com/tianxinzizhen/templatedb/util"
)

type AnyDB interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

type SelectDB[T any] struct {
	db       *DefaultDB
	selectdb AnyDB
}

func newScanDest(columns []*sql.ColumnType) (map[string]int, []any) {
	indexMap := make(map[string]int, len(columns))
	retSlice := make([]any, len(columns))
	for i, item := range columns {
		v := reflect.New(item.ScanType())
		indexMap[item.Name()] = i
		retSlice[i] = v.Interface()
	}
	return indexMap, retSlice
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

func (any *SelectDB[T]) newReceiver(len int) *T {
	dest := new(T)
	t := util.RefType(reflect.TypeOf(dest))
	if util.RefType(t).Kind() == reflect.Map {
		*dest = reflect.MakeMapWithSize(reflect.MapOf(t.Key(), t.Elem()), len).Interface().(T)
	} else if util.RefType(t).Kind() == reflect.Slice {
		*dest = reflect.MakeSlice(reflect.SliceOf(t.Elem()), 0, len).Interface().(T)
	}
	return dest
}

func (any *SelectDB[T]) query(params any, name []any) (*sql.Rows, []*sql.ColumnType, error) {
	statement := getSkipFuncName(3, name)
	sql, args, err := any.db.templateBuild(statement, params)
	if err != nil {
		return nil, nil, err
	}
	rows, err := any.selectdb.Query(sql, args...)
	if err != nil {
		return nil, nil, err
	}
	columns, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, err
	}
	return rows, columns, nil
}

func (any *SelectDB[T]) Select(params any, name ...any) (rowSlice []*T, err error) {
	rows, columns, err := any.query(params, name)
	if err != nil {
		return
	}
	defer rows.Close()
	scanIndex, scanSlice := newScanDest(columns)
	ret := *(new([]*T))
	for rows.Next() {
		err = rows.Scan(scanSlice...)
		if err != nil {
			return
		}
		receiver := any.newReceiver(len(columns))
		util.ConvertResultAnys(columns, scanIndex, scanSlice, receiver, template.AsTagString)
		ret = append(ret, receiver)
	}
	return ret, nil
}
func (any *SelectDB[T]) SelectFirst(params any, name ...any) (row *T, err error) {
	rows, columns, err := any.query(params, name)
	if err != nil {
		return
	}
	defer rows.Close()
	scanIndex, scanSlice := newScanDest(columns)
	var receiver *T
	if rows.Next() {
		err = rows.Scan(scanSlice...)
		if err != nil {
			return
		}
		receiver = any.newReceiver(len(columns))
		util.ConvertResultAnys(columns, scanIndex, scanSlice, receiver, template.AsTagString)
	}
	return receiver, nil
}
