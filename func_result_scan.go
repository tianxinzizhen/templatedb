package templatedb

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/tianxinzizhen/templatedb/scanner"
	"github.com/tianxinzizhen/templatedb/template"
)

func newScanDestByValues(sqlParamType map[reflect.Type]struct{}, columns []*sql.ColumnType, ret []reflect.Value) (destSlice []any, more bool, arrayLen int, err error) {
	if len(ret) == 0 {
		return nil, false, 0, fmt.Errorf("not scan dest")
	}
	destSlice = make([]any, 0, len(columns))
	if len(ret) == 1 {
		t := ret[0].Type()
		if t.Kind() == reflect.Array || t.Kind() == reflect.Slice {
			if t.Kind() == reflect.Array {
				arrayLen = t.Len()
			}
			t = t.Elem()
			more = true
		}
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
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
					Column:  columns[0],
					Convert: scanConvertByDatabaseType[columns[0].DatabaseTypeName()],
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
					Column:  columns[0],
					Convert: scanConvertByDatabaseType[columns[0].DatabaseTypeName()],
				})
			}
		}
	}
	return destSlice, more, arrayLen, nil
}

func nextScan(ret []reflect.Value, rowi int, scanRows []any) {
	if len(ret) == 1 {
		rt := ret[0].Type()
		var more bool
		var isArray = rt.Kind() == reflect.Array
		if rt.Kind() == reflect.Array || rt.Kind() == reflect.Slice {
			rt = rt.Elem()
			more = true
		}
		rv := reflect.New(rt).Elem()
		for rv.Kind() == reflect.Pointer {
			rt = rt.Elem()
			rv.Set(reflect.New(rt))
			rv = rv.Elem()
		}
		if more {
			rt := ret[0].Type().Elem()
			mv := rv
			for rt.Kind() == reflect.Pointer {
				rt = rt.Elem()
				mv = rv.Addr()
			}
			if isArray {
				if ret[0].IsZero() {
					ret[0] = reflect.New(ret[0].Type()).Elem()
				}
				ret[0].Index(rowi).Set(mv)
			} else {
				ret[0] = reflect.Append(ret[0], mv)
			}
		} else {
			mv := rv
			rt := ret[0].Type()
			for rt.Kind() == reflect.Pointer {
				rt = rt.Elem()
				mv = rv.Addr()
			}
			ret[0] = mv
		}
		if _, ok := sqlParamType[rt]; rt.Kind() == reflect.Struct && !ok {
			for _, v := range scanRows {
				if vi, ok := v.(*scanner.StructScanner); ok {
					vi.Dest = rv
				}
			}
		} else if rt.Kind() == reflect.Map && rt.Key().Kind() == reflect.String {
			rv.Set(reflect.MakeMapWithSize(reflect.MapOf(rt.Key(), rt.Elem()), len(scanRows)))
			for _, v := range scanRows {
				if vi, ok := v.(*scanner.MapScanner); ok {
					vi.Dest = rv
				}
			}
		} else if rt.Kind() == reflect.Slice {
			rv.Set(reflect.MakeSlice(reflect.SliceOf(rt.Elem()), len(scanRows), len(scanRows)))
			for _, v := range scanRows {
				if vi, ok := v.(*scanner.SliceScanner); ok {
					vi.Dest = rv
				}
			}
		} else {
			for _, v := range scanRows {
				if vi, ok := v.(*scanner.ParameterScanner); ok {
					ret[0] = reflect.New(ret[0].Type()).Elem()
					vi.Dest = ret[0]
				}
			}
		}
	} else {
		for i, v := range scanRows {
			if vi, ok := v.(*scanner.ParameterScanner); ok {
				ret[i] = reflect.New(ret[i].Type()).Elem()
				vi.Dest = ret[i]
			}
		}
	}
}
