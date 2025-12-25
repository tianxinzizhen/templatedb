package templatedb

import (
	"database/sql"
	"fmt"
	"reflect"
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
func (tdb *DBFuncTemplateDB) getScanDest(columns []*sql.ColumnType, ret []reflect.Value) (destSlice []any, err error) {
	if len(ret) == 0 {
		return nil, fmt.Errorf("not scan dest")
	}
	var t reflect.Type
	if len(ret) == 1 {
		t = ret[0].Type()
		var v reflect.Value
		switch t.Kind() {
		case reflect.Pointer:
			v = reflect.New(t.Elem())
			ret[0] = v
			v = v.Elem()
		case reflect.Map:
			if t.Key().Kind() != reflect.String {
				return nil, fmt.Errorf("map key must be string")
			}
			if t.Elem() != reflect.TypeOf((*any)(nil)).Elem() {
				return nil, fmt.Errorf("map value must not be pointer")
			}
			v = reflect.MakeMap(t)
			ret[0] = v
		case reflect.Struct:
			v = reflect.New(t).Elem()
			ret[0] = v
		case reflect.Slice:
			t = t.Elem()
			if t.Kind() == reflect.Pointer {
				v = reflect.New(t.Elem())
				ret[0] = reflect.Append(ret[0], v)
				v = v.Elem()
			} else {
				v = reflect.New(t).Elem()
				ret[0] = reflect.Append(ret[0], v)
			}
		}
		for _, c := range columns {
			switch v.Type().Kind() {
			case reflect.Struct:
				fname := tdb.filedName(v.Type(), c.Name())
				fv := v.FieldByName(fname)
				if fv.IsValid() && fv.CanSet() {
					destSlice = append(destSlice, fv.Addr().Interface())
				} else {
					destSlice = append(destSlice, getTempScanDest(c.ScanType()))
				}
			case reflect.Map:
				val := reflect.New(v.Type().Elem())
				v.SetMapIndex(reflect.ValueOf(c.Name()), val)
				destSlice = append(destSlice, val.Interface())
			default:
				if v.CanSet() {
					destSlice = append(destSlice, v.Addr().Interface())
				} else {
					destSlice = append(destSlice, getTempScanDest(c.ScanType()))
				}
			}
		}
	} else {
		if len(columns) > 0 {
			for i := 0; i < len(ret); i++ {
				ret[i] = reflect.New(ret[i].Type()).Elem()
			}
			for i := 0; i < len(columns); i++ {
				if i < len(ret) {
					if ret[i].CanSet() {
						destSlice = append(destSlice, ret[i].Addr().Interface())
					} else {
						destSlice = append(destSlice, getTempScanDest(columns[i].ScanType()))
					}
				} else {
					destSlice = append(destSlice, getTempScanDest(columns[i].ScanType()))
				}
			}
		}
	}
	return
}
