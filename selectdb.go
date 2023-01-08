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

func (any *SelectDB[T]) newModel(len int) *T {
	dest := new(T)
	t := util.RefType(reflect.TypeOf(dest))
	if util.RefType(t).Kind() == reflect.Map {
		*dest = reflect.MakeMapWithSize(reflect.MapOf(t.Key(), t.Elem()), len).Interface().(T)
	} else if util.RefType(t).Kind() == reflect.Slice {
		*dest = reflect.MakeSlice(reflect.SliceOf(t.Elem()), 0, len).Interface().(T)
	}
	return dest
}

func (any *SelectDB[T]) Query(statement string, params any) (*sql.Rows, []*sql.ColumnType, error) {
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

func (any *SelectDB[T]) Select(statement string, params any) (rowSlice []*T, err error) {
	rows, columns, err := any.Query(statement, params)
	if err != nil {
		return
	}
	defer rows.Close()
	ret := *(new([]*T))
	for rows.Next() {
		receiver := any.newModel(len(columns))
		destIndex, retSlice := newScanDest(columns)
		err = rows.Scan(retSlice...)
		if err != nil {
			return
		}
		util.ConvertResultAnys(columns, destIndex, retSlice, receiver, template.AsTagString)
		ret = append(ret, receiver)
	}
	return ret, nil
}
func (any *SelectDB[T]) SelectFirst(statement string, params any) (row *T, err error) {
	rows, columns, err := any.Query(statement, params)
	if err != nil {
		return
	}
	defer rows.Close()
	var receiver *T
	if rows.Next() {
		receiver = any.newModel(len(columns))
		destIndex, retSlice := newScanDest(columns)
		err = rows.Scan(retSlice...)
		if err != nil {
			return
		}
		util.ConvertResultAnys(columns, destIndex, retSlice, receiver, template.AsTagString)
	}
	return receiver, nil
}
