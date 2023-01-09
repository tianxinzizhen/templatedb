package util

import (
	"database/sql"
	"reflect"
	"strings"
	"time"
)

// 获取引用类型
func RefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface {
		t = t.Elem()
	}
	return t
}

func RefValue(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Type().Kind() == reflect.Ptr || v.Type().Kind() == reflect.Interface) {
		v = v.Elem()
	}
	return v
}

func setFieldMap(dest any, as string) map[string]reflect.Value {
	paramMap := make(map[string]reflect.Value)
	setFields(dest, paramMap, as)
	return paramMap
}
func setFields(dest any, paramMap map[string]reflect.Value, as string) {
	t := RefType(reflect.TypeOf(dest))
	v := RefValue(reflect.ValueOf(dest))
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			fv := v.Field(i)
			fName := sf.Name
			if asName, ok := sf.Tag.Lookup(as); ok {
				if asName == "-" {
					continue
				}
				fName, _, _ = strings.Cut(asName, ",")
			}
			paramMap[fName] = fv
		}
	}
}

func getValue(scanVal any) any {
	switch v := scanVal.(type) {
	case *sql.NullBool:
		if v.Valid {
			return v.Bool
		} else {
			return false
		}
	case *sql.NullByte:
		if v.Valid {
			return v.Byte
		} else {
			return 0
		}
	case *sql.NullFloat64:
		if v.Valid {
			return v.Float64
		} else {
			return float64(0)
		}
	case *sql.NullInt16:
		if v.Valid {
			return v.Int16
		} else {
			return int16(0)
		}
	case *sql.NullInt32:
		if v.Valid {
			return v.Int32
		} else {
			return int32(0)
		}
	case *sql.NullInt64:
		if v.Valid {
			return v.Int64
		} else {
			return int64(0)
		}
	case *sql.NullString:
		if v.Valid {
			return v.String
		} else {
			return ""
		}
	case *sql.NullTime:
		if v.Valid {
			return v.Time
		} else {
			return time.Time{}
		}
	default:
		return reflect.ValueOf(v).Elem().Interface()
	}
}

func ConvertResultAnys(columns []*sql.ColumnType, scanIndex map[string]int, scanSlice []any, receiver any, as string) {
	t := RefType(reflect.TypeOf(receiver))
	if t.Kind() == reflect.Struct {
		valueMap := setFieldMap(receiver, as)
		for _, item := range columns {
			if v, ok := valueMap[item.Name()]; ok {
				index := scanIndex[item.Name()]
				scanVal := scanSlice[index]
				rScanVal := reflect.ValueOf(getValue(scanVal))
				if rScanVal.CanConvert(v.Type()) {
					v.Set(rScanVal.Convert(v.Type()))
				}
			}
		}
	} else if t.Kind() == reflect.Map {
		v := RefValue(reflect.ValueOf(receiver))
		if t.Key().Kind() == reflect.String {
			for _, item := range columns {
				index := scanIndex[item.Name()]
				scanVal := scanSlice[index]
				rScanVal := reflect.ValueOf(getValue(scanVal))
				if rScanVal.CanConvert(t.Elem()) {
					v.SetMapIndex(reflect.ValueOf(item.Name()), rScanVal.Convert(t.Elem()))
				} else {
					v.SetMapIndex(reflect.ValueOf(item.Name()), reflect.New(t.Elem()).Elem())
				}
			}
		}
	} else if t.Kind() == reflect.Slice {
		sliceVal := RefValue(reflect.ValueOf(receiver))
		for index := range columns {
			rScanVal := reflect.ValueOf(getValue(scanSlice[index]))
			if rScanVal.CanConvert(t.Elem()) {
				scanSlice[index] = rScanVal.Convert(t.Elem()).Interface()
			} else {
				scanSlice[index] = reflect.New(t.Elem()).Elem().Interface()
			}
		}
		for _, v := range scanSlice {
			sliceVal.Set(reflect.Append(sliceVal, reflect.ValueOf(v)))
		}
	} else {
		v := RefValue(reflect.ValueOf(receiver))
		rScanVal := reflect.ValueOf(getValue(scanSlice[0]))
		if rScanVal.CanConvert(t) {
			v.Set(rScanVal.Convert(t))
		}
	}
}
