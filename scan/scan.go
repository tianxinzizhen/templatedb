package scan

import (
	"database/sql"
	"reflect"
)

type ScanVal interface {
	sql.Scanner
	Val() any
	ValPtr() any
}

var localScanVal = make(map[reflect.Type]reflect.Type)

func RegisterScanVal(sv ScanVal) {
	localScanVal[reflect.TypeOf(sv.Val())] = reflect.TypeOf(sv).Elem()
	localScanVal[reflect.TypeOf(sv.ValPtr())] = reflect.TypeOf(sv).Elem()
}

func isScanVal(t reflect.Type) bool {
	_, ok := localScanVal[t]
	return ok
}

func isNotScanVal(t reflect.Type) bool {
	return !isScanVal(t)
}

func getScanValType(t reflect.Type) reflect.Type {
	if sv, ok := localScanVal[t]; ok {
		return sv
	}
	return t
}

func getScanVal(v reflect.Value) reflect.Value {
	sv := v
	if v.CanAddr() && v.Kind() == reflect.Struct {
		sv = sv.Addr()
	}
	if svv, ok := sv.Interface().(ScanVal); ok {
		return reflect.ValueOf(svv.Val())
	}
	return v
}

func getScanValPtr(v reflect.Value) reflect.Value {
	sv := v
	if v.CanAddr() && v.Kind() == reflect.Struct {
		sv = sv.Addr()
	}
	if svv, ok := sv.Interface().(ScanVal); ok {
		return reflect.ValueOf(svv.ValPtr())
	}
	return v
}
