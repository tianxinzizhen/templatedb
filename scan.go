package templatedb

import (
	"reflect"
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
