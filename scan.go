package templatedb

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/tianxinzizhen/templatedb/scanner"
	"github.com/tianxinzizhen/templatedb/template"
)

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

var scanConvertByDatabaseType map[string]func(field reflect.Value, v any) error = make(map[string]func(field reflect.Value, v any) error)

func AddScanConvertDatabaseTypeFunc(key string, funcMethod func(field reflect.Value, v any) error) {
	scanConvertByDatabaseType[key] = funcMethod
}

func newScanDest(columns []*sql.ColumnType, t reflect.Type) ([]any, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	destSlice := make([]any, 0, len(columns))
	if _, ok := sqlParamType[t]; t.Kind() == reflect.Struct && !ok {
		scanMapIndex := make(map[string]int)
		for _, item := range columns {
			f, ok := template.GetFieldByName(t, item.Name(), scanMapIndex)
			if ok {
				destSlice = append(destSlice, &scanner.StructScanner{Convert: scanConvertByDatabaseType[item.DatabaseTypeName()], Index: f.Index})
			} else {
				destSlice = append(destSlice, getTempScanDest(item.ScanType()))
			}
		}
		return destSlice, nil
	} else if t.Kind() == reflect.Map {
		if t.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("scan map key type not string")
		}
		for _, v := range columns {
			destSlice = append(destSlice, &scanner.MapScanner{Column: v, Name: v.Name()})
		}
		return destSlice, nil
	} else if t.Kind() == reflect.Slice {
		for i, v := range columns {
			destSlice = append(destSlice, &scanner.SliceScanner{Column: v, Index: i})
		}
		return destSlice, nil
	} else if t.Kind() == reflect.Func {
		if t.NumIn() == 0 && t.NumOut() > 0 {
			i := 0
			for ; i < t.NumOut(); i++ {
				destSlice = append(destSlice, &scanner.ParameterScanner{Column: columns[i], Convert: scanConvertByDatabaseType[columns[i].DatabaseTypeName()]})
			}
			for ; i < len(columns); i++ {
				destSlice = append(destSlice, getTempScanDest(columns[i].ScanType()))
			}
			return destSlice, nil
		} else if t.NumOut() == 0 && t.NumIn() > 0 {
			i := 0
			for ; i < t.NumIn(); i++ {
				destSlice = append(destSlice, &scanner.ParameterScanner{Column: columns[i], Convert: scanConvertByDatabaseType[columns[i].DatabaseTypeName()]})
			}
			for ; i < len(columns); i++ {
				destSlice = append(destSlice, getTempScanDest(columns[i].ScanType()))
			}
			return destSlice, nil
		} else {
			return nil, fmt.Errorf("scan func In(%d) Out(%d) not supported", t.NumIn(), t.NumOut())
		}
	} else {
		if len(columns) > 0 {
			destSlice = append(destSlice, &scanner.ParameterScanner{Column: columns[0], Convert: scanConvertByDatabaseType[columns[0].DatabaseTypeName()]})
			for i := 1; i < len(columns); i++ {
				destSlice = append(destSlice, getTempScanDest(columns[i].ScanType()))
			}
		}
		return destSlice, nil
	}
}

func newReceiver(t reflect.Type, columns []*sql.ColumnType, scanRows []any) reflect.Value {
	ret := reflect.New(t).Elem()
	val := ret
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
		ret.Set(reflect.New(t))
		ret = ret.Elem()
	}
	if t.Kind() == reflect.Struct {
		for _, v := range scanRows {
			if vi, ok := v.(*scanner.StructScanner); ok {
				vi.Dest = ret
			}
		}
	} else if t.Kind() == reflect.Map && t.Key().Kind() == reflect.String {
		dest := reflect.MakeMapWithSize(reflect.MapOf(t.Key(), t.Elem()), len(columns))
		for _, v := range scanRows {
			if vi, ok := v.(*scanner.MapScanner); ok {
				vi.Dest = dest
			}
		}
	} else if t.Kind() == reflect.Slice {
		dest := reflect.MakeSlice(reflect.SliceOf(t.Elem()), len(columns), len(columns))
		for _, v := range scanRows {
			if vi, ok := v.(*scanner.SliceScanner); ok {
				vi.Dest = dest
			}
		}
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
				if vi, ok := v.(*scanner.ParameterScanner); ok {
					vi.Dest = results[i]
				}
			}
			ret.Set(dest)
		} else {
			var results []reflect.Value = make([]reflect.Value, 0, t.NumIn())
			for i := 0; i < t.NumIn(); i++ {
				results = append(results, reflect.New(t.In(i)).Elem())
			}
			for i, v := range scanRows {
				if vi, ok := v.(*scanner.ParameterScanner); ok {
					vi.Dest = results[i]
				}
			}
			return reflect.ValueOf(results)
		}
	} else {
		for _, v := range scanRows {
			if vi, ok := v.(*scanner.ParameterScanner); ok {
				vi.Dest = ret
			}
		}
	}
	return val
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
	dest, err := newScanDest(columns, st)
	if err != nil {
		return reflect.Zero(t).Interface().(T), err
	}
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
	dest, err := newScanDest(columns, t)
	if err != nil {
		return reflect.Zero(t).Interface().(T), err
	}
	receiver := newReceiver(t, columns, dest)
	err = rows.Scan(dest...)
	if err != nil {
		return reflect.Zero(t).Interface().(T), err
	}
	return receiver.Interface().(T), nil
}
