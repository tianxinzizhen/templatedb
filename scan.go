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

func newScanDest(columns []*sql.ColumnType, t reflect.Type) []any {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	destSlice := make([]any, 0, len(columns))
	if t.Kind() == reflect.Struct {
		indexMap := make(map[int][]int, len(columns))
		scanMapIndex := make(map[string]int)
		for i, item := range columns {
			f, ok := template.GetFieldByName(t, item.Name(), scanMapIndex)
			if ok {
				indexMap[i] = f.Index
			} else {
				panic(fmt.Errorf("类型%v无法扫描字段：%s", t, item.Name()))
			}
		}
		for si, v := range columns {
			destSlice = append(destSlice, &scanner.StructScanner{Convert: scanConvertByDatabaseType[v.DatabaseTypeName()], Index: indexMap[si]})
		}
		return destSlice
	} else if t.Kind() == reflect.Map {
		if t.Key().Kind() != reflect.String {
			panic(fmt.Errorf("scan map key type not string"))
		}
		for _, v := range columns {
			destSlice = append(destSlice, &scanner.MapScanner{Column: v, Name: v.Name()})
		}
		return destSlice
	} else if t.Kind() == reflect.Slice {
		for i, v := range columns {
			destSlice = append(destSlice, &scanner.SliceScanner{Column: v, Index: i})
		}
		return destSlice
	} else if t.Kind() == reflect.Func {
		if t.NumIn() == 0 && t.NumOut() > 0 {
			i := 0
			for ; i < t.NumOut(); i++ {
				destSlice = append(destSlice, &scanner.ParameterScanner{Column: columns[i], Convert: scanConvertByDatabaseType[columns[i].DatabaseTypeName()]})
			}
			for ; i < len(columns); i++ {
				destSlice = append(destSlice, getTempScanDest(columns[i].ScanType()))
			}
			return destSlice
		} else if t.NumOut() == 0 && t.NumIn() > 0 {
			i := 0
			for ; i < t.NumIn(); i++ {
				destSlice = append(destSlice, &scanner.ParameterScanner{Column: columns[i], Convert: scanConvertByDatabaseType[columns[i].DatabaseTypeName()]})
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
			destSlice = append(destSlice, &scanner.ParameterScanner{Column: columns[0], Convert: scanConvertByDatabaseType[columns[0].DatabaseTypeName()]})
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
			if vi, ok := v.(*scanner.StructScanner); ok {
				vi.Dest = dv
			}
		}
	} else if t.Kind() == reflect.Map && t.Key().Kind() == reflect.String {
		dest := reflect.MakeMapWithSize(reflect.MapOf(t.Key(), t.Elem()), len(columns))
		for _, v := range scanRows {
			if vi, ok := v.(*scanner.MapScanner); ok {
				vi.Dest = dest
			}
		}
		ret.Elem().Set(dest)
	} else if t.Kind() == reflect.Slice {
		dest := reflect.MakeSlice(reflect.SliceOf(t.Elem()), len(columns), len(columns))
		for _, v := range scanRows {
			if vi, ok := v.(*scanner.SliceScanner); ok {
				vi.Dest = dest
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
				if vi, ok := v.(*scanner.ParameterScanner); ok {
					vi.Dest = results[i]
				}
			}
			ret.Elem().Set(dest)
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
		dest := ret.Elem()
		for _, v := range scanRows {
			if vi, ok := v.(*scanner.ParameterScanner); ok {
				vi.Dest = dest
			}
		}
	}
	if rt.Kind() == reflect.Pointer {
		return ret
	} else {
		return ret.Elem()
	}
}
