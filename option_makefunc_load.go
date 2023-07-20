package templatedb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"github.com/tianxinzizhen/templatedb/util"
)

type LoadType int

const (
	LoadXML LoadType = iota
	LoadComment
)

func DBFuncInitCommon[T any](tdb TemplateOptionDB, dbFuncStruct *T) (reflect.Value, error) {
	dv, isNil := util.Indirect(reflect.ValueOf(dbFuncStruct))
	if isNil {
		return reflect.Value{}, errors.New("InitMakeFunc In(0) is nil")
	}
	dt := dv.Type()
	if dt.Kind() != reflect.Struct {
		return reflect.Value{}, errors.New("InitMakeFunc In(0) type is not struct")
	}
	if !dv.FieldByName("DBFunc").IsValid() {
		return reflect.Value{}, errors.New("strcut type need anonymous templatedb.DBFunc")
	}
	dv.FieldByName("Begin").Set(reflect.ValueOf(func() (*T, error) {
		if db, ok := tdb.(*OptionDB); ok {
			tx, err := db.Begin()
			if err != nil {
				return nil, err
			}
			nt := new(T)
			DBFuncInit(nt, tx)
			return nt, nil
		} else {
			return nil, fmt.Errorf("Begin error: Currently in a transactional state")
		}
	}))
	dv.FieldByName("BeginTx").Set(reflect.ValueOf(func(ctx context.Context, opts *sql.TxOptions) (*T, error) {
		if db, ok := tdb.(*OptionDB); ok {
			tx, err := db.BeginTx(ctx, opts)
			if err != nil {
				return nil, err
			}
			nt := new(T)
			DBFuncInit(nt, tx)
			return nt, nil
		} else {
			return nil, fmt.Errorf("BeginTx error: Currently in a transactional state")
		}
	}))
	if tx, ok := tdb.(*OptionTxDB); ok {
		dv.FieldByName("AutoCommit").Set(reflect.ValueOf(tx.AutoCommit))
	} else {
		dv.FieldByName("AutoCommit").Set(reflect.ValueOf(func(ctx context.Context, errp *error) {}))
	}
	return dv, nil
}

func DBFuncInitAndLoad[T any](tdb *OptionDB, dbFuncStruct *T, sql any, lt LoadType) (*T, error) {
	dv, err := DBFuncInitCommon(tdb, dbFuncStruct)
	if err != nil {
		return nil, err
	}
	dt := dv.Type()
	pkg := dt.PkgPath()
	//添加数据信息
	switch lt {
	case LoadXML:
		err := tdb.LoadXml(pkg, sql)
		if err != nil {
			return nil, err
		}
	case LoadComment:
		err := tdb.LoadComment(pkg, sql)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("DBFuncInitAndLoad not load sql script data")
	}
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
