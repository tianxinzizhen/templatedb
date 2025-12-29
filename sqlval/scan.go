package sqlval

import (
	"database/sql"
	"fmt"
	"reflect"
)

type ScanVal[T any] interface {
	sql.Scanner
	Val() T
	ValPtr() *T
}

var localScanVal = make(map[reflect.Type]reflect.Type)

func RegisterScanVal[T any](sv ScanVal[T]) error {
	if reflect.TypeFor[T]().Kind() == reflect.Pointer {
		return fmt.Errorf("sv.Val() must be not pointer")
	}
	if _, ok := localScanVal[reflect.TypeFor[T]()]; ok {
		return fmt.Errorf("sv.Val() type already registered")
	}
	if _, ok := localScanVal[reflect.TypeFor[*T]()]; ok {
		return fmt.Errorf("sv.ValPtr() type already registered")
	}
	localScanVal[reflect.TypeFor[T]()] = reflect.TypeOf(sv).Elem()
	localScanVal[reflect.TypeFor[*T]()] = reflect.TypeOf(sv).Elem()
	return nil
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
	method := v.MethodByName("Val")
	if method.IsValid() {
		return method.Call([]reflect.Value{})[0]
	}
	return v
}

func getScanValPtr(v reflect.Value) reflect.Value {
	method := v.MethodByName("ValPtr")
	if method.IsValid() {
		return method.Call([]reflect.Value{})[0]
	}
	return v
}
