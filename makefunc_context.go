package tgsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"github.com/tianxinzizhen/tgsql/load"
	"github.com/tianxinzizhen/tgsql/template"
)

func handleParam(sqlInfo *load.SqlDataInfo, op *funcExecOption, args []reflect.Value) {
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
}

func makeDBFuncContext(t reflect.Type, tdb *TgenSql, action Operation, templateSql *template.Template, sqlInfo *load.SqlDataInfo) reflect.Value {
	return reflect.MakeFunc(t, func(args []reflect.Value) (results []reflect.Value) {
		var err error
		var hasReturnErr bool
		if t.NumOut() > 0 {
			hasReturnErr = t.Out(t.NumOut() - 1).Implements(errorType)
		}
		op := &funcExecOption{
			ctx: context.Background(), // default ctx
		}
		// 处理参数
		handleParam(sqlInfo, op, args)
		handleErr := func() {
			if hasReturnErr {
				results[t.NumOut()-1] = reflect.ValueOf(funcErr(sqlInfo.FuncName, err))
			} else {
				tdb.enableRecover(op.ctx)
				panic(recoverLog(err))
			}
		}
		if !GetEnableSqlTx(op.ctx) {
			var conn *sql.Conn
			conn, err = tdb.db.Conn(op.ctx)
			if err != nil {
				handleErr()
				return results
			}
			defer conn.Close()
			op.db = conn
		}
		if sqlInfo.NotPrepare {
			op.option |= optionNotPrepare
		}
		// 批量执行
		if sqlInfo.BatchInsert {
			op.option |= optionBatchInsert
		}
		results = make([]reflect.Value, t.NumOut())
		for i := 0; i < t.NumOut(); i++ {
			results[i] = reflect.Zero(t.Out(i))
		}
		var changeOp func(op *funcExecOption) (bool, error)
		if op.option&optionBatchInsert != 0 {
			if action != ExecNoResultAction {
				err = errors.New("batch insert only support exec no result action")
				handleErr()
				return results
			}
			pv := reflect.ValueOf(op.param)
			switch pv.Kind() {
			case reflect.Slice:
				if pv.Len() == 0 {
					err = errors.New("batch insert param is empty")
				}
			default:
				err = errors.New("batch insert param type not support")
			}
			if err != nil {
				handleErr()
				return results
			}
			changeOp = func(op *funcExecOption) (bool, error) {
				if op.offset < pv.Len() {
					op.param = pv.Index(op.offset).Interface()
					op.offset++
					err = tdb.templateBuild(templateSql, op)
					return true, nil
				}
				return false, nil
			}
		} else {
			err = tdb.templateBuild(templateSql, op)
		}
		if err != nil {
			handleErr()
			return results
		}
		switch action {
		case ExecAction:
			var ret sql.Result
			ret, err = tdb.exec(op)
			if err != nil {
				handleErr()
				return results
			}
			if ret != nil {
				result := reflect.ValueOf(ret)
				for i := 0; i < t.NumOut(); i++ {
					if t.Out(i) == sqlResultType {
						results[i] = result
					}
				}
			}
		case SelectAction:
			op.result = results
			if hasReturnErr {
				op.result = results[:len(results)-1]
			}
			err = tdb.query(op)
			if err != nil {
				handleErr()
				return results
			}
		case SelectOneAction:
			op.result = results
			if hasReturnErr {
				op.result = results[:len(results)-1]
			}
			err = tdb.queryOption(op, queryOption{selectOne: true})
			if err != nil {
				handleErr()
				return results
			}
		case ExecNoResultAction:
			if op.option&optionBatchInsert != 0 {
				stmtMap := map[string]*sql.Stmt{}
				var preSql string
				defer func() {
					for _, s := range stmtMap {
						if s != nil {
							s.Close()
						}
					}
				}()
				for {
					var change bool
					change, err = changeOp(op)
					if err != nil {
						handleErr()
						return results
					}
					if preSql != op.sql {
						if _, ok := stmtMap[preSql]; !ok {
							var stmt *sql.Stmt
							stmt, err = tdb.prepareContext(op)
							if err != nil {
								handleErr()
								return results
							}
							preSql = op.sql
							stmtMap[preSql] = stmt
							op.stmt = stmt
						}
					}
					if stmt, ok := stmtMap[op.sql]; ok {
						op.stmt = stmt
						_, err = stmt.ExecContext(op.ctx, op.args...)
						if err != nil {
							handleErr()
							return results
						}
					}
					if !change {
						break
					}
				}
			} else {
				_, err = tdb.exec(op)
				if err != nil {
					handleErr()
					return results
				}
			}
		}
		return results
	})
}

func checkAllDBFuncSet(tdb *TgenSql, dv reflect.Value) error {
	for dv.Kind() == reflect.Pointer {
		dv = dv.Elem()
	}
	if !dv.IsValid() {
		return errors.New("checkAllDBFuncSet in(dbFuncStruct) is not valid")
	}
	dt := dv.Type()
	tgsv := reflect.ValueOf(tdb)
	for i := 0; i < dv.NumField(); i++ {
		f := dv.Field(i)
		ft := f.Type()
		if ft.Kind() == reflect.Func {
			if f.IsNil() {
				return fmt.Errorf("%s method:%s is not have a sql statement", dt.Name(), dt.Field(i).Name)
			}
		} else if ft == tgsv.Type() {
			f.Set(tgsv)
		}
	}
	return nil
}

func InitDBFunc(tdb *TgenSql, dest any) error {
	dv := reflect.ValueOf(dest)
	for dv.Kind() == reflect.Pointer {
		dv = dv.Elem()
	}
	if !dv.IsValid() {
		return errors.New("NewDBFunc in(dbFuncStruct) is not valid")
	}
	dt := dv.Type()
	tp := template.New(dt.Name()).Delims(tdb.leftDelim, tdb.rightDelim).
		Funcs(tdb.sqlFunc)
	fkey := fmt.Sprintf("%s.%s", dt.PkgPath(), dt.Name())
	sqlInfos := tdb.localFuncDataInfo.GetSqlDataInfo(fkey)
	if len(sqlInfos) == 0 {
		return fmt.Errorf("NewDBFunc not found sql script data in type %s", dt.Name())
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
						return fmt.Errorf("NewDBFunc in(%d) type not support %s", i, ditIni.Kind().String())
					}
				}
				for i := 0; i < fct.NumOut(); i++ {
					ditIni := fct.Out(i)
					if ditIni.Implements(errorType) {
						continue
					}
					switch ditIni.Kind() {
					case reflect.Func, reflect.Chan:
						return fmt.Errorf("NewDBFunc out(%d) type not support %s", i, ditIni.Kind().String())
					case reflect.Interface:
						if !ditIni.Implements(sqlResultType) {
							return fmt.Errorf("NewDBFunc out(%d) type not support Interface", i)
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
	err := checkAllDBFuncSet(tdb, dv)
	if err != nil {
		return err
	}
	return nil
}
