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
		op := &funcExecOption{}
		if sqlInfo.NotPrepare {
			op.option |= OptionNotPrepare
		}
		// 批量执行
		if sqlInfo.Batch {
			op.option |= OptionBatch
		}
		var opArgs []any
		var useMultiParam bool
		for _, v := range args {
			val := v.Interface()
			opArgs = append(opArgs, val)
			if v.Type().Implements(contextType) {
				if val != nil {
					op.ctx = val.(context.Context)
				}
			} else {
				pvt := v.Type()
				if pvt.Kind() == reflect.Pointer {
					pvt = pvt.Elem()
				}
				switch pvt.Kind() {
				case reflect.Map, reflect.Slice, reflect.Struct:
					if op.param == nil {
						op.param = val
					} else {
						useMultiParam = true
					}
				default:
					useMultiParam = true
				}
			}
		}
		if useMultiParam && len(sqlInfo.Param) > 0 {
			paramMap := map[string]any{}
			for i, v := range sqlInfo.Param {
				if reflect.ValueOf(opArgs[i]).Type().Implements(contextType) {
					continue
				}
				paramMap[v] = opArgs[i]
			}
			op.param = paramMap
		}
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
				tdb.enableRecover(op.ctx)
				panic(recoverLog(err))
			}
			return results
		}
		switch action {
		case ExecAction:
			var ret sql.Result
			ret, err = tdb.exec(db, op)
			if ret != nil {
				result := reflect.ValueOf(ret)
				if t.Out(0).Kind() == reflect.Interface {
					results[0] = result
				}
			}
		case SelectAction:
			op.result = results
			if hasReturnErr {
				op.result = results[:len(results)-1]
			}
			err = tdb.query(db, op)
		case SelectOneAction:
			op.result = results
			if hasReturnErr {
				op.result = results[:len(results)-1]
			}
			err = tdb.queryOption(db, op, queryOption{selectOne: true})
		case ExecNoResultAction:
			_, err = tdb.exec(db, op)
		}
		if err != nil {
			if hasReturnErr {
				results[t.NumOut()-1] = reflect.ValueOf(funcErr(sqlInfo.FuncName, err))
			} else {
				tdb.enableRecover(op.ctx)
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
	tp := template.New(dt.Name()).Delims(tdb.leftDelim, tdb.rightDelim).
		Funcs(tdb.sqlFunc)
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
		_, err := tp.AddParse(sqlInfo.Name, sqlInfo.Sql)
		if err != nil {
			return err
		}
		if fc, ok := dt.FieldByName(sqlInfo.Name); ok {
			t := tp.Lookup(sqlInfo.Name)
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
					switch ditIni.Kind() {
					case reflect.Func, reflect.Chan:
						return fmt.Errorf("DBFuncContextInit in(%d) type not support %s", i, ditIni.Kind().String())
					}
				}
				for i := 0; i < fct.NumOut(); i++ {
					ditIni := fct.Out(i)
					if ditIni.Implements(errorType) {
						continue
					}
					switch ditIni.Kind() {
					case reflect.Func, reflect.Chan:
						return fmt.Errorf("DBFuncContextInit out(%d) type not support %s", i, ditIni.Kind().String())
					case reflect.Interface:
						if !ditIni.Implements(sqlResultType) {
							return fmt.Errorf("DBFuncContextInit out(%d) type not support Interface", i)
						}
					}
				}
				var action Operation = ExecNoResultAction
				if fct.NumOut() > 0 {
					if fct.Out(0) == sqlResultType {
						action = ExecAction
					} else if !fct.Out(0).Implements(errorType) {
						if fct.Out(0).Kind() == reflect.Slice {
							action = SelectAction
						} else {
							action = SelectOneAction
						}
					}
				}
				fcv.Set(makeDBFuncContext(fct, tdb, action, t, sqlInfo))
			}
		}
	}
	//check the method of initialization
	for i := 0; i < dv.NumField(); i++ {
		f := dv.Field(i)
		ft := f.Type()
		if ft.Kind() == reflect.Func {
			if f.IsNil() {
				return fmt.Errorf("%s method:%s is not have a sql statement", dt.Name(), dt.Field(i).Name)
			}
		}
	}
	return nil
}
