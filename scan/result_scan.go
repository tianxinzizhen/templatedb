package scan

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
		return tempScanDest[scanType]
	} else {
		return dest
	}
}

func setMapValue(t reflect.Type, ret []reflect.Value, isSlice bool) (v reflect.Value, err error) {
	if t.Key().Kind() != reflect.String {
		err = fmt.Errorf("map key must be string")
		return
	}
	v = reflect.MakeMap(t)
	if isSlice {
		ret[0] = reflect.Append(ret[0], v)
	} else {
		ret[0] = v
	}
	return
}

func setValue(t reflect.Type, ret []reflect.Value, isSlice bool) (v reflect.Value, deferFn []func()) {
	if isScanVal(t) {
		v = reflect.New(getScanValType(t)).Elem()
		deferFn = append(deferFn, func() {
			if isSlice {
				switch t.Kind() {
				case reflect.Pointer:
					ret[0] = reflect.Append(ret[0], getScanValPtr(v))
				default:
					ret[0] = reflect.Append(ret[0], getScanVal(v))
				}
			} else {
				switch t.Kind() {
				case reflect.Pointer:
					ret[0] = getScanValPtr(v)
				default:
					ret[0] = getScanVal(v)
				}
			}
		})
	} else {
		switch t.Kind() {
		case reflect.Pointer:
			v = reflect.New(t.Elem())
		default:
			v = reflect.New(t).Elem()
		}
		if t.Kind() == reflect.Pointer {
			if isSlice {
				ret[0] = reflect.Append(ret[0], v)
			} else {
				ret[0] = v
			}
			v = v.Elem()
		} else {
			if isSlice {
				deferFn = append(deferFn, func() {
					ret[0] = reflect.Append(ret[0], v)
				})
			} else {
				deferFn = append(deferFn, func() {
					ret[0] = v
				})
			}
		}
	}
	return
}
func GetScanDest(filedName func(t reflect.Type, name string) string, columns []*sql.ColumnType, ret []reflect.Value) (destSlice []any, deferFn []func(), err error) {
	if len(ret) == 0 {
		err = fmt.Errorf("not scan dest")
		return
	}
	var t reflect.Type
	if len(ret) == 1 {
		t = ret[0].Type()
		var v reflect.Value
		var df []func()
		switch t.Kind() {
		case reflect.Map:
			v, err = setMapValue(t, ret, false)
			if err != nil {
				return
			}
		case reflect.Slice:
			t = t.Elem()
			switch t.Kind() {
			case reflect.Map:
				v, err = setMapValue(t, ret, true)
				if err != nil {
					return
				}
			default:
				v, df = setValue(t, ret, true)
			}
		default:
			v, df = setValue(t, ret, false)
		}
		isOne := true
		for _, c := range columns {
			switch v.Type().Kind() {
			case reflect.Map:
				valT := v.Type().Elem()
				val := reflect.New(valT).Elem()
				deferFn = append(deferFn, func() {
					v.SetMapIndex(reflect.ValueOf(c.Name()), val)
				})
				destSlice = append(destSlice, val.Addr().Interface())
			case reflect.Struct:
				if isNotScanVal(v.Type()) {
					fname := filedName(v.Type(), c.Name())
					fv := v.FieldByName(fname)
					if !fv.IsValid() || !fv.CanSet() {
						destSlice = append(destSlice, getTempScanDest(c.ScanType()))
						continue
					}
					if isScanVal(fv.Type()) {
						scanV := reflect.New(getScanValType(fv.Type())).Elem()
						destSlice = append(destSlice, scanV.Addr().Interface())
						deferFn = append(deferFn, func() {
							switch fv.Kind() {
							case reflect.Pointer:
								fv.Set(getScanValPtr(scanV))
							default:
								fv.Set(getScanVal(scanV))
							}
						})
					} else {
						destSlice = append(destSlice, fv.Addr().Interface())
					}
					break
				}
				fallthrough
			default:
				if isOne && v.CanSet() {
					isOne = false
					destSlice = append(destSlice, v.Addr().Interface())
				} else {
					destSlice = append(destSlice, getTempScanDest(c.ScanType()))
				}
			}
		}
		deferFn = append(deferFn, df...)
	} else {
		if len(columns) > 0 {
			for i := 0; i < len(columns); i++ {
				if i < len(ret) {
					t := ret[i].Type()
					if isScanVal(t) {
						scanV := reflect.New(getScanValType(t)).Elem()
						destSlice = append(destSlice, scanV.Addr().Interface())
						deferFn = append(deferFn, func() {
							switch t.Kind() {
							case reflect.Pointer:
								ret[i] = getScanValPtr(scanV)
							default:
								ret[i] = getScanVal(scanV)
							}
						})
					} else {
						ret[i] = reflect.New(t).Elem()
						destSlice = append(destSlice, ret[i].Addr().Interface())
					}
				} else {
					destSlice = append(destSlice, getTempScanDest(columns[i].ScanType()))
				}
			}
		}
	}
	return
}
