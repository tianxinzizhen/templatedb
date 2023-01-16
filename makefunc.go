package templatedb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"github.com/tianxinzizhen/templatedb/util"
)

var (
	contextType       = reflect.TypeOf((*context.Context)(nil)).Elem()
	ResultType        = reflect.TypeOf(Result{})
	PrepareResultType = reflect.TypeOf(PrepareResult{})
)

type Operation int

const (
	ExecAction Operation = iota
	PrepareAction
	SelectAction
	SelectScanAction
	ExecNoResultAction
)

type DBFunc[T any] struct {
	Begin      func() (*T, error)
	BeginTx    func(ctx context.Context, opts *sql.TxOptions) (*T, error)
	AutoCommit func(errp *error)
	Recover    func(errp *error)
}

type Result struct {
	LastInsertId int64
	RowsAffected int64
}

type PrepareResult struct {
	RowsAffected int64
}

// 自动初始化构造方法
func DBFuncInit[T any](dbfuncStruct *T, tdb TemplateDB) error {
	dv, isNil := util.Indirect(reflect.ValueOf(dbfuncStruct))
	if isNil {
		return errors.New("InitMakeFunc In(0) is nil")
	}
	dt := dv.Type()
	if dt.Kind() != reflect.Struct {
		return errors.New("InitMakeFunc In(0) type is not struct")
	}
	if !dv.FieldByName("DBFunc").IsValid() {
		return errors.New("strcut type need anonymous templatedb.DBFunc")
	}
	dv.FieldByName("Begin").Set(reflect.ValueOf(func() (*T, error) {
		if db, ok := tdb.(*DefaultDB); ok {
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
		if db, ok := tdb.(*DefaultDB); ok {
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
	if tx, ok := tdb.(*TemplateTxDB); ok {
		dv.FieldByName("AutoCommit").Set(reflect.ValueOf(tx.AutoCommit))
	} else {
		dv.FieldByName("AutoCommit").Set(reflect.ValueOf(func(errp *error) {}))
	}
	if db, ok := tdb.(*DefaultDB); ok {
		dv.FieldByName("Recover").Set(reflect.ValueOf(db.Recover))
	} else {
		dv.FieldByName("Recover").Set(reflect.ValueOf(func(errp *error) {}))
	}
	for i := 0; i < dt.NumField(); i++ {
		dist := dt.Field(i)
		dit := dist.Type
		div := dv.Field(i)
		if dit.Kind() == reflect.Func {
			switch dit.NumIn() {
			case 0, 1:
			case 2:
				if !(dit.In(0).Implements(contextType) || dit.In(1).Implements(contextType)) {
					return fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] Func In types is not correct", dt.PkgPath(), dt.Name(), dist.Name)
				}
			default:
				return fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] Func In type is not correct", dt.PkgPath(), dt.Name(), dist.Name)
			}
			if dit.NumOut() == 1 {
				switch dit.Out(0) {
				case ResultType:
					div.Set(makeDBFunc(dit, tdb, ExecAction, fmt.Sprintf("%s.%s", dt.PkgPath(), dt.Name()), dist.Name))
				case PrepareResultType:
					div.Set(makeDBFunc(dit, tdb, PrepareAction, fmt.Sprintf("%s.%s", dt.PkgPath(), dt.Name()), dist.Name))
				default:
					div.Set(makeDBFunc(dit, tdb, SelectAction, fmt.Sprintf("%s.%s", dt.PkgPath(), dt.Name()), dist.Name))
				}
			} else if dit.NumOut() == 0 {
				if dit.NumIn() > 0 && dit.In(dit.NumIn()-1).Kind() == reflect.Func {
					div.Set(makeDBFunc(dit, tdb, SelectScanAction, fmt.Sprintf("%s.%s", dt.PkgPath(), dt.Name()), dist.Name))
				} else {
					div.Set(makeDBFunc(dit, tdb, ExecNoResultAction, fmt.Sprintf("%s.%s", dt.PkgPath(), dt.Name()), dist.Name))
				}
			} else {
				return fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] Func In and Out type is not correct", dt.PkgPath(), dt.Name(), dist.Name)
			}
		}
	}
	return nil
}

func makeDBFunc(t reflect.Type, tdb TemplateDB, action Operation, pkg, fieldName string) reflect.Value {
	return reflect.MakeFunc(t, func(args []reflect.Value) (results []reflect.Value) {
		var ctx context.Context
		var param any
		for _, v := range args {
			if v.Type().Implements(contextType) {
				ctx = v.Interface().(context.Context)
			} else {
				param = v.Interface()
			}
		}
		if ctx == nil {
			ctx = context.Background()
		}
		switch action {
		case ExecAction:
			lastInsertId, rowsAffected := tdb.ExecContext(ctx, param, pkg, fieldName)
			result := reflect.ValueOf(&Result{LastInsertId: lastInsertId, RowsAffected: rowsAffected})
			if t.Out(0).Kind() == reflect.Pointer {
				return []reflect.Value{result}
			} else {
				return []reflect.Value{result.Elem()}
			}
		case PrepareAction:
			pv := reflect.ValueOf(param)
			var pvs []any
			if pv.IsValid() && (pv.Kind() == reflect.Slice || pv.Kind() == reflect.Array) {
				for i := 0; i < pv.Len(); i++ {
					pvs = append(pvs, pv.Index(i).Interface())
				}
			}
			rowsAffected := tdb.PrepareExecContext(ctx, pvs, pkg, fieldName)
			prepareResult := reflect.ValueOf(&PrepareResult{RowsAffected: rowsAffected})
			if t.Out(0).Kind() == reflect.Pointer {
				return []reflect.Value{prepareResult}
			} else {
				return []reflect.Value{prepareResult.Elem()}
			}
		case SelectAction:
			return []reflect.Value{tdb.selectByType(ctx, param, t.Out(0), pkg, fieldName)}
		case SelectScanAction:
			tdb.SelectScanFuncContext(ctx, param, param, pkg, fieldName)
			return nil
		case ExecNoResultAction:
			tdb.ExecContext(ctx, param, pkg, fieldName)
			return nil
		default:
			return nil
		}
	})
}
