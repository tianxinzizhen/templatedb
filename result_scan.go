package templatedb

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/tianxinzizhen/templatedb/scanner"
)

var tempScanDest map[reflect.Type]any

// 创建临时扫描字段
func getTempScanDest(scanType reflect.Type) any {
	if tempScanDest == nil {
		tempScanDest = make(map[reflect.Type]any)
	}
	if dest, ok := tempScanDest[scanType]; !ok {
		tempScanDest[scanType] = reflect.New(scanType).Interface()
		return dest
	} else {
		return dest
	}
}
func (tdb *DBFuncTemplateDB) newScanDestByValues(columns []*sql.ColumnType, ret []reflect.Value) (destSlice []any, more bool, arrayLen int, err error) {
	if len(ret) == 0 {
		return nil, false, 0, fmt.Errorf("not scan dest")
	}
	destSlice = make([]any, 0, len(columns))
	if len(ret) == 1 {
		t := ret[0].Type()
		switch t.Kind() {
		case reflect.Array:
			arrayLen = t.Len()
			more = true
		case reflect.Slice:
			more = true
		}
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
			switch t.Kind() {
			case reflect.Array, reflect.Slice:
				err = fmt.Errorf("scan array or slice type not support")
				return
			}
		}
		if _, ok := tdb.getParameterMap[t]; t.Kind() == reflect.Struct && !ok {
			scanMapIndex := make(map[string]int)
			for _, item := range columns {
				f, ok := DefaultGetFieldByName(t, item.Name(), scanMapIndex)
				if ok {
					destSlice = append(destSlice, &scanner.StructScanner{
						Index:        f.Index,
						SetParameter: tdb.setParameterMap[f.Type],
					})
				} else {
					destSlice = append(destSlice, getTempScanDest(item.ScanType()))
				}
			}
		} else if t.Kind() == reflect.Map {
			if t.Key().Kind() != reflect.String {
				return nil, false, 0, fmt.Errorf("scan map key type not string")
			}
			for _, v := range columns {
				destSlice = append(destSlice, &scanner.MapScanner{Column: v, Name: v.Name()})
			}
		} else if t.Kind() == reflect.Slice {
			for i, v := range columns {
				destSlice = append(destSlice, &scanner.SliceScanner{Column: v, Index: i})
			}
		} else {
			if len(columns) > 0 {
				destSlice = append(destSlice, &scanner.ParameterScanner{
					Column:       columns[0],
					SetParameter: tdb.setParameterMap[ret[0].Type()],
				})
				for i := 1; i < len(columns); i++ {
					destSlice = append(destSlice, getTempScanDest(columns[i].ScanType()))
				}
			}
		}
	} else {
		if len(columns) > 0 {
			for i := 0; i < len(columns); i++ {
				destSlice = append(destSlice, &scanner.ParameterScanner{
					Column:       columns[i],
					SetParameter: tdb.setParameterMap[ret[i].Type()],
				})
			}
		}
	}
	return destSlice, more, arrayLen, nil
}

func (tdb *DBFuncTemplateDB) nextNewScanDest(ret []reflect.Value, scanRows []any) {
	if len(ret) == 1 {
		rt := ret[0].Type()
		if rt.Kind() == reflect.Array || rt.Kind() == reflect.Slice {
			rt = rt.Elem()
		}
		rv := reflect.New(rt).Elem()
		for rv.Kind() == reflect.Pointer {
			rt = rt.Elem()
			rv.Set(reflect.New(rt))
			rv = rv.Elem()
		}
		switch rt.Kind() {
		case reflect.Slice:
			rv.Set(reflect.MakeSlice(reflect.SliceOf(rt.Elem()), len(scanRows), len(scanRows)))
		case reflect.Map:
			rv.Set(reflect.MakeMap(reflect.MapOf(rt.Key(), rt.Elem())))
		}
		for _, v := range scanRows {
			if vi, ok := v.(scanner.Scanner); ok {
				vi.SetDest(rv)
			}
		}
	} else {
		for i, v := range scanRows {
			if vi, ok := v.(scanner.Scanner); ok {
				vi.SetDest(reflect.New(ret[i].Type()).Elem())
			}
		}
	}
}

func nextSetResult(ret []reflect.Value, rowi int, scanRows []any) {
	if len(ret) == 1 {
		rt := ret[0].Type()
		var more bool
		switch rt.Kind() {
		case reflect.Array, reflect.Slice:
			more = true
		}
		var rv reflect.Value
		for _, v := range scanRows {
			if vi, ok := v.(scanner.Scanner); ok {
				rv = vi.GetDest()
				break
			}
		}
		if !rv.IsValid() {
			return
		}
		if more {
			rt = rt.Elem()
			mv := rv
			for rt.Kind() == reflect.Pointer {
				rt = rt.Elem()
				mv = rv.Addr()
			}
			if rt.Kind() == reflect.Array {
				if ret[0].IsZero() {
					ret[0] = reflect.New(ret[0].Type()).Elem()
				}
				ret[0].Index(rowi).Set(mv)
			} else {
				ret[0] = reflect.Append(ret[0], mv)
			}
		} else {
			mv := rv
			for rt.Kind() == reflect.Pointer {
				rt = rt.Elem()
				mv = rv.Addr()
			}
			ret[0] = mv
		}
	} else {
		for i, v := range scanRows {
			if vi, ok := v.(scanner.Scanner); ok {
				ret[i] = vi.GetDest()
			}
		}
	}
}
