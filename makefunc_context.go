package templatedb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/tianxinzizhen/templatedb/load"
	"github.com/tianxinzizhen/templatedb/template"
)

type keySqlTx struct{}

func NewSqlTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, keySqlTx{}, tx)
}

func FromSqlTx(ctx context.Context) (tx *sql.Tx, ok bool) {
	tx, ok = ctx.Value(keySqlTx{}).(*sql.Tx)
	return
}

func recoverLog(err error) *DBFuncPanicError {
	if err != nil {
		var pc []uintptr = make([]uintptr, MaxStackLen)
		n := runtime.Callers(3, pc[:])
		frames := runtime.CallersFrames(pc[:n])
		sb := strings.Builder{}
		sb.WriteString(fmt.Sprintf("%v \n", err))
		for frame, more := frames.Next(); more; frame, more = frames.Next() {
			sb.WriteString(fmt.Sprintf("%s:%d \n", frame.File, frame.Line))
		}
		msg := sb.String()
		return &DBFuncPanicError{msg: msg, err: err}
	}
	return nil
}
func funcErr(funcName string, err error) *DBFuncError {
	if err != nil {
		var pc []uintptr = make([]uintptr, 2)
		n := runtime.Callers(3, pc)
		frames := runtime.CallersFrames(pc[:n])
		var msg string
		for frame, more := frames.Next(); more; frame, more = frames.Next() {
			msg = fmt.Sprintf("%s:%d", frame.File, frame.Line)
		}
		return &DBFuncError{funcName: funcName, funcFileLine: msg, err: err}
	}
	return nil
}

type DBFuncPanicError struct {
	msg string
	err error
}

func (e *DBFuncPanicError) Error() string {
	return e.msg
}

func (e *DBFuncPanicError) Unwrap() error {
	return e.err
}

type DBFuncError struct {
	funcName     string
	funcFileLine string
	err          error
}

func (e *DBFuncError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s FuncName:%s An error has occurred [%s]", e.funcFileLine, e.funcName, e.err.Error())
	}
	return e.funcName
}

func (e *DBFuncError) Unwrap() error {
	return e.err
}

func makeDBFuncContext(t reflect.Type, tdb *DBFuncTemplateDB, action Operation, templateSql *template.Template, sqlInfo *load.SqlDataInfo) reflect.Value {
	return reflect.MakeFunc(t, func(args []reflect.Value) (results []reflect.Value) {
		op := &FuncExecOption{}
		var opArgs []any
		for _, v := range args {
			val := v.Interface()
			op.args_Index = append(op.args_Index, val)
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

		var db sqlDB = tdb.db
		if op.ctx == nil {
			op.ctx = context.Background()
		} else {
			tx, ok := FromSqlTx(op.ctx)
			if ok && tx != nil {
				db = tx
			}
		}
		results = make([]reflect.Value, t.NumOut())
		for i := 0; i < t.NumOut(); i++ {
			results[i] = reflect.Zero(t.Out(i))
		}
		var hasReturnErr bool
		if t.NumOut() > 0 {
			hasReturnErr = t.Out(t.NumOut() - 1).Implements(errorType)
		}
		var err error
		err = tdb.templateBuild(templateSql, op)
		if err != nil {
			if hasReturnErr {
				results[t.NumOut()-1] = reflect.ValueOf(funcErr(sqlInfo.FuncName, err))
			} else {
				panic(recoverLog(err))
			}
			return results
		}
		switch action {
		case ExecAction:
			var ret *Result
			ret, err = tdb.exec(db, op)
			if ret != nil {
				result := reflect.ValueOf(ret)
				if t.Out(0).Kind() == reflect.Pointer {
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
				results[t.NumOut()-1] = reflect.ValueOf(funcErr(sqlInfo.FuncName, err))
			} else {
				panic(recoverLog(err))
			}
		}
		return results
	})
}

func DBFuncContextInit(tdb *DBFuncTemplateDB, dbFuncStruct any, lt LoadType, sql any) error {
	dv := reflect.ValueOf(dbFuncStruct)
	for dv.Kind() == reflect.Pointer {
		dv = dv.Elem()
	}
	if !dv.IsValid() {
		return errors.New("DBFuncContextInit in(dbFuncStruct) is not valid")
	}
	dt := dv.Type()
	tp := template.New(dt.Name()).Delims(tdb.leftDelim, tdb.rightDelim).SqlParams(tdb.sqlParamsConvert).Funcs(tdb.sqlFunc)
	var sqlInfos []*load.SqlDataInfo
	var err error
	//添加数据信息
	switch lt {
	case LoadXML:
		sqlInfos, err = load.LoadXml(dt.PkgPath(), sql)
		if err != nil {
			return err
		}
	case LoadComment:
		sqlInfos, err = load.LoadComment(dt.PkgPath(), sql)
		if err != nil {
			return err
		}
	default:
		return errors.New("DBFuncContextInit not load sql script data")
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
		t.Param = sqlInfo.Param
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
					if ditIni.Implements(errorType) {
						continue
					}
					if ditIni.Kind() == reflect.Func {
						return fmt.Errorf("DBFuncContextInit out(%d) type not support Func", i)
					}
					if ditIni.Kind() == reflect.Chan {
						return fmt.Errorf("DBFuncContextInit out(%d) type not support Chan", i)
					}
					if ditIni.Kind() == reflect.Interface {
						return fmt.Errorf("DBFuncContextInit out(%d) type not support Interface", i)
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
				fcv.Set(makeDBFuncContext(fct, tdb, action, t, sqlInfo))
			}
		}
	}
	return nil
}

func AutoCommit(ctx context.Context, err *error) {
	tx, ok := FromSqlTx(ctx)
	if ok && tx != nil {
		if *err != nil {
			tx.Rollback()
			return
		}
		if e := recover(); e != nil {
			tx.Rollback()
			switch e := e.(type) {
			case error:
				*err = e
			default:
				panic(e)
			}
			return
		}
		tx.Commit()
	}
}

func Recover(_ context.Context, err *error) {
	if *err == nil {
		if e := recover(); e != nil {
			switch e := e.(type) {
			case error:
				*err = e
			default:
				panic(e)
			}
			return
		}
	}
}
