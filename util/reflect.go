package util

import (
	"reflect"
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
