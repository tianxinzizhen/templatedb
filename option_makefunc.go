package templatedb

import (
	"context"
	"fmt"
	"reflect"
)

func DBFuncInit[T any](dbFuncStruct *T, tdb TemplateOptionDB) (*T, error) {
	dv, err := DBFuncInitCommon(tdb, dbFuncStruct)
	if err != nil {
		return nil, err
	}
	dt := dv.Type()
	for i := 0; i < dt.NumField(); i++ {
		dist := dt.Field(i)
		dit := dist.Type
		div := dv.Field(i)
		if dit.Kind() == reflect.Func {
			var action Operation = ExecNoResultAction
			for i := 0; i < dit.NumIn(); i++ {
				ditIni := dit.In(i)
				if ditIni.Implements(contextType) {
					continue
				}
				if ditIni.Kind() == reflect.Pointer {
					ditIni = ditIni.Elem()
				}
				if ditIni.Kind() == reflect.Func {
					action = SelectScanAction
					break
				}
			}
			if dit.NumOut() > 0 {
				if dit.Out(0) == ResultType || dit.Out(0) == ResultType.Elem() {
					action = ExecAction
				} else if !dit.Out(0).Implements(errorType) {
					action = SelectAction
				}
			}
			div.Set(makeDBFunc(dit, tdb, action, fmt.Sprintf("%s.%s.%s", dt.PkgPath(), dt.Name(), dist.Name)))
		}
	}
	return dbFuncStruct, nil
}

func makeDBFunc(t reflect.Type, tdb TemplateOptionDB, action Operation, funcName string) reflect.Value {
	return reflect.MakeFunc(t, func(args []reflect.Value) (results []reflect.Value) {
		op := NewExecOption()
		var opArgs []any
		for _, v := range args {
			if v.Type().Implements(contextType) {
				op.SetContext(v.Interface().(context.Context))
			} else if v.Type().Kind() == reflect.Func {
				op.SetResult(v.Interface())
			} else {
				pvt := v.Type()
				if pvt.Kind() == reflect.Pointer {
					pvt = pvt.Elem()
				}
				switch pvt.Kind() {
				case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct:
					if _, ok := sqlParamType[pvt]; ok {
						opArgs = append(opArgs, v.Interface())
					} else {
						op.SetParam(v.Interface())
					}
				default:
					opArgs = append(opArgs, v.Interface())
				}
			}
		}
		op.SetArgs(opArgs...)
		op.SetFuncName(funcName)
		if op.Ctx == nil {
			op.Ctx = context.Background()
		}
		op.Ctx = context.WithValue(op.Ctx, TemplateDBFuncName, funcName)
		results = make([]reflect.Value, t.NumOut())
		if t.NumOut() == 2 || (t.NumOut() == 1 && t.Out(0).Implements(errorType)) {
			defer func() {
				e := recover()
				if e != nil {
					switch rerr := e.(type) {
					case error:
						if t.NumOut() == 2 {
							results[0] = reflect.Zero(t.Out(0))
						}
						results[len(results)-1] = reflect.ValueOf(rerr)
						recoverPrintf(op.Ctx, rerr)
					default:
						panic(e)
					}
				}
			}()
			results[len(results)-1] = reflect.Zero(errorType)
		}
		switch action {
		case ExecAction:
			lastInsertId, rowsAffected := tdb.TExec(op)
			result := reflect.ValueOf(&Result{LastInsertId: lastInsertId, RowsAffected: rowsAffected})
			if t.Out(0).Kind() == reflect.Pointer {
				results[0] = result
			} else {
				results[0] = result.Elem()
			}
		case SelectAction:
			op.SetResult(reflect.Zero(t.Out(0)).Interface())
			results[0] = reflect.ValueOf(tdb.TQuery(op))
		case SelectScanAction:
			tdb.TQuery(op)
		case ExecNoResultAction:
			tdb.TExec(op)
		}
		return results
	})
}
