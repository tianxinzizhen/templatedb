package sqlval

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
)

type Convert[T any] interface {
	ConvertValue(v T) (any, error)
	ConvertValuePtr(v *T) (any, error)
}

var localConvertValMap = make(map[reflect.Type]reflect.Value)

func RegisterConvert[T any](gv Convert[T]) error {
	if reflect.TypeFor[T]().Kind() == reflect.Pointer {
		return fmt.Errorf("gv.ConvertValuePtr() must be not pointer")
	}
	localConvertValMap[reflect.TypeFor[T]()] = reflect.ValueOf(gv.ConvertValue)
	localConvertValMap[reflect.TypeFor[*T]()] = reflect.ValueOf(gv.ConvertValuePtr)
	return nil
}

func localConvertVal(v reflect.Value) []reflect.Value {
	if method, ok := localConvertValMap[v.Type()]; ok {
		if method.IsValid() {
			return method.Call([]reflect.Value{v})
		}
	} else {
		jv := v
		for jv.Kind() == reflect.Pointer {
			jv = jv.Elem()
		}
		switch jv.Kind() {
		case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
			if !jv.IsValid() {
				return []reflect.Value{v, reflect.ValueOf(nil)}
			}
			mJson, err := json.Marshal(jv.Interface())
			if err != nil {
				return []reflect.Value{v, reflect.ValueOf(err)}
			}
			return []reflect.Value{reflect.ValueOf(string(mJson)), reflect.ValueOf(nil)}
		}
	}
	return []reflect.Value{v, reflect.ValueOf(nil)}
}

func ConvertValue(ci any, v any) (any, error) {
	var err error
	nvc, _ := ci.(driver.NamedValueChecker)
	if nvc != nil {
		nv := &driver.NamedValue{
			Value: v,
		}
		err = nvc.CheckNamedValue(nv)
		if err == nil {
			return nv.Value, nil
		}
	}
	if err != nil {
		v, err = driver.DefaultParameterConverter.ConvertValue(v)
	}
	if err != nil {
		ret := localConvertVal(reflect.ValueOf(v))
		if ret[1].IsValid() {
			return ret[0].Interface(), ret[1].Interface().(error)
		}
		return ret[0].Interface(), nil
	}
	return v, nil
}

func ConvertValues(ci any, args []any) ([]any, error) {
	ret := []any{}
	for _, arg := range args {
		v, err := ConvertValue(ci, arg)
		if err != nil {
			return nil, err
		}
		ret = append(ret, v)
	}
	return ret, nil
}
