package templatedb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"github.com/tianxinzizhen/templatedb/load"
	"github.com/tianxinzizhen/templatedb/template"
)

type DBFuncContextTx struct{}

func NewDBFuncContextTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, DBFuncContextTx{}, tx)
}

func FromDBFuncContextTx(ctx context.Context) (tx *sql.Tx, ok bool) {
	tx, ok = ctx.Value(DBFuncContextTx{}).(*sql.Tx)
	return
}

func AutoCommitFromContext(ctx context.Context, errp *error) {
	if *errp == nil {
		e := recover()
		if e != nil {
			switch err := e.(type) {
			case error:
				*errp = err
			default:
				panic(e)
			}
		}
	}
	tx, ok := FromDBFuncContextTx(ctx)
	if ok && tx != nil {
		if *errp != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}
}

func makeDBFuncContext(t reflect.Type, tdb *DBFuncTemplateDB, action Operation, templateSql *template.Template) reflect.Value {
	return reflect.MakeFunc(t, func(args []reflect.Value) (results []reflect.Value) {
		op := &FuncExecOption{
			args_Index: map[int]any{},
		}
		var opArgs []any
		for i, v := range args {
			val := v.Interface()
			op.args_Index[i] = val
			if v.Type().Implements(contextType) {
				op.ctx = val.(context.Context)
			} else {
				pvt := v.Type()
				if pvt.Kind() == reflect.Pointer {
					pvt = pvt.Elem()
				}
				switch pvt.Kind() {
				case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct:
					if _, ok := tdb.sqlParamType[pvt]; ok {
						opArgs = append(opArgs, val)
					} else {
						op.param = val
					}
				default:
					opArgs = append(opArgs, val)
				}
			}
		}
		op.args = opArgs
		results = make([]reflect.Value, t.NumOut())
		for i := 0; i < t.NumOut(); i++ {
			results[i] = reflect.Zero(t.Out(i))
		}
		hasReturnErr := t.Out(t.NumOut() - 1).Implements(errorType)
		var err error
		err = tdb.templateBuild(templateSql, op)
		if err != nil {
			if hasReturnErr {
				results[t.NumOut()-1].Set(reflect.ValueOf(err))
			} else {
				panic(err)
			}
			return results
		}
		var db sqlDB = tdb.db
		if op.ctx == nil {
			op.ctx = context.Background()
		} else {
			tx, ok := FromDBFuncContextTx(op.ctx)
			if ok && tx != nil {
				db = tx
			}
			op.ctx = context.WithValue(op.ctx, TemplateDBFuncName, templateSql.Name)
		}
		switch action {
		case ExecAction:
			var ret *Result
			ret, err = tdb.exec(db, op)
			if ret != nil {
				result := reflect.ValueOf(ret)
				if t.Out(0).Kind() != reflect.Pointer {
					results[0] = result
				} else {
					results[0] = result.Elem()
				}
			}
		case SelectAction:
			op.result = results
			if hasReturnErr {
				op.result = results[:len(results)-1]
			}
			err = tdb.query(db, op)
		case ExecNoResultAction:
			_, err = tdb.exec(db, op)
		}
		if err != nil {
			if hasReturnErr {
				results[t.NumOut()-1].Set(reflect.ValueOf(err))
			} else {
				panic(err)
			}
		}
		return results
	})
}

func DBFuncContextInit(tdb *DBFuncTemplateDB, dbFuncStruct any, sql any) error {
	dv := reflect.ValueOf(dbFuncStruct)
	for dv.Kind() == reflect.Pointer {
		dv = dv.Elem()
	}
	if !dv.IsValid() {
		return errors.New("DBFuncContextInit in(dbFuncStruct) is not valid")
	}
	dt := dv.Type()
	tp := template.New(dt.Name()).Delims(tdb.leftDelim, tdb.rightDelim).SqlParams(tdb.sqlParamsConvert).Funcs(sqlFunc)
	sqlInfos, err := load.LoadComment(sql)
	if err != nil {
		return err
	}
	for _, sqlInfo := range sqlInfos {
		_, err = tp.ParseName(sqlInfo.Name, sqlInfo.Sql)
		if err != nil {
			return err
		}
		if sqlInfo.Common {
			continue
		}
		t := tp.Lookup(sqlInfo.Name)
		t.NotPrepare = sqlInfo.NotPrepare
		t.ParamMap = sqlInfo.ParamMap
		if fc, ok := dt.FieldByName(sqlInfo.Name); ok {
			fct := fc.Type
			if fct.Kind() == reflect.Func {
				fcv := dv.FieldByIndex(fc.Index)
				for i := 0; i < fct.NumIn(); i++ {
					ditIni := fct.In(i)
					if ditIni.Implements(contextType) {
						continue
					}
					if ditIni.Kind() == reflect.Pointer {
						ditIni = ditIni.Elem()
					}
					if ditIni.Kind() == reflect.Func {
						return fmt.Errorf("DBFuncContextInit in(%d) type not support Func", i)
					}
					if ditIni.Kind() == reflect.Chan {
						return fmt.Errorf("DBFuncContextInit in(%d) type not support Chan", i)
					}
				}
				for i := 0; i < fct.NumOut(); i++ {
					ditIni := fct.Out(i)
					if ditIni.Kind() == reflect.Func {
						return fmt.Errorf("DBFuncContextInit out(%d) type not support Func", i)
					}
					if ditIni.Kind() == reflect.Chan {
						return fmt.Errorf("DBFuncContextInit out(%d) type not support Chan", i)
					}
				}
				var action Operation = ExecNoResultAction
				if fct.NumOut() > 0 {
					if fct.Out(0) == ResultType || fct.Out(0) == ResultType.Elem() {
						action = ExecAction
					} else if !fct.Out(0).Implements(errorType) {
						action = SelectAction
					}
				}
				fcv.Set(makeDBFuncContext(fct, tdb, action, t))
			}
		}
	}
	return nil
}
