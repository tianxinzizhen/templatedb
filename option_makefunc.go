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
		for i, v := range args {
			val := v.Interface()
			op.args_Index[i] = val
			if v.Type().Implements(contextType) {
				op.SetContext(val.(context.Context))
			} else if v.Type().Kind() == reflect.Func {
				op.SetResult(val)
			} else {
				pvt := v.Type()
				if pvt.Kind() == reflect.Pointer {
					pvt = pvt.Elem()
				}
				switch pvt.Kind() {
				case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct:
					if _, ok := sqlParamType[pvt]; ok {
						opArgs = append(opArgs, val)
					} else {
						op.SetParam(val)
					}
				default:
					opArgs = append(opArgs, val)
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
		for i := 0; i < t.NumOut(); i++ {
			results[i] = reflect.Zero(t.Out(i))
		}
		hasReturnErr := t.Out(t.NumOut() - 1).Implements(errorType)
		if hasReturnErr {
			defer func() {
				e := recover()
				if e != nil {
					switch rerr := e.(type) {
					case error:
						results[len(results)-1] = reflect.ValueOf(rerr)
						recoverPrintf(op.Ctx, rerr)
					default:
						panic(e)
					}
				}
			}()
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
			sr := results
			if hasReturnErr {
				sr = sr[:len(sr)-1]
			}
			if len(sr) == 1 {
				op.SetResult(sr[0].Interface())
				sr[0] = reflect.ValueOf(tdb.TQuery(op))
			} else {
				out := make([]reflect.Type, 0, len(sr))
				for _, v := range sr {
					out = append(out, v.Type())
				}
				ft := reflect.FuncOf(nil, out, false)
				op.SetResult(reflect.Zero(ft).Interface())
				if fn := reflect.ValueOf(tdb.TQuery(op)); fn.IsValid() && !fn.IsNil() {
					copy(sr, fn.Call(nil))
				}
			}
		case SelectScanAction:
			tdb.TQuery(op)
		case ExecNoResultAction:
			tdb.TExec(op)
		}
		return results
	})
}
