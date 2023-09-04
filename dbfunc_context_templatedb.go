package templatedb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"runtime"

	"github.com/tianxinzizhen/templatedb/template"
)

type DBFuncTemplateDB struct {
	db                    *sql.DB
	leftDelim, rightDelim string
	sqlParamsConvert      func(val reflect.Value) (string, any)
	sqlDebug              bool
	logFunc               func(ctx context.Context, info string)
	sqlParamType          map[reflect.Type]struct{}
	sqlFunc               template.FuncMap
	template              map[uintptr]map[int]*template.Template
}

func (tdb *DBFuncTemplateDB) Delims(leftDelim, rightDelim string) {
	tdb.leftDelim = leftDelim
	tdb.rightDelim = rightDelim
}

func (tdb *DBFuncTemplateDB) SqlParamsConvert(sqlParamsConvert func(val reflect.Value) (string, any)) {
	tdb.sqlParamsConvert = sqlParamsConvert
}

func (tdb *DBFuncTemplateDB) SqlDebug(sqlDebug bool) {
	tdb.sqlDebug = sqlDebug
}

func (tdb *DBFuncTemplateDB) LogFunc(logFunc func(ctx context.Context, info string)) {
	tdb.logFunc = logFunc
}

func (tdb *DBFuncTemplateDB) AddSqlParamType(t reflect.Type) {
	tdb.sqlParamType[t] = struct{}{}
}
func (tdb *DBFuncTemplateDB) AddTemplateFunc(key string, funcMethod any) error {
	if _, ok := sqlFunc[key]; ok {
		return fmt.Errorf("add template func[%s] already exists ", key)
	} else {
		tdb.sqlFunc[key] = funcMethod
	}
	return nil
}
func NewDBFuncTemplateDB(sqlDB *sql.DB) *DBFuncTemplateDB {
	tdb := &DBFuncTemplateDB{
		db:        sqlDB,
		leftDelim: "{", rightDelim: "}",
		sqlParamType: make(map[reflect.Type]struct{}),
		sqlFunc:      make(template.FuncMap),
	}
	for k, v := range sqlParamType {
		tdb.sqlParamType[k] = v
	}
	for k, v := range sqlFunc {
		tdb.sqlFunc[k] = v
	}
	tdb.logFunc = LogPrintf
	return tdb
}

type funcExecOption struct {
	ctx        context.Context
	param      any
	args       []any
	args_Index []any
	result     []reflect.Value
	sql        string
}

type keyLogSqlFuncName struct{}

func FromLogSqlFuncName(ctx context.Context) (sql string, ok bool) {
	sql, ok = ctx.Value(keyLogSqlFuncName{}).(string)
	return
}

func (tdb *DBFuncTemplateDB) templateBuild(templateSql *template.Template, op *funcExecOption) error {
	var err error
	op.sql, op.args, err = templateSql.ExecuteBuilder(op.param, op.args, op.args_Index)
	if err != nil {
		return err
	}
	if templateSql.NotPrepare {
		op.sql, err = SqlInterpolateParams(op.sql, op.args)
		if err != nil {
			return err
		}
		op.args = nil
	}
	if tdb.sqlDebug && tdb.logFunc != nil {
		interpolateParamsSql, err := SqlInterpolateParams(op.sql, op.args)
		ctx := context.WithValue(op.ctx, keyLogSqlFuncName{}, templateSql.Name())
		if err != nil {
			tdb.logFunc(ctx, fmt.Sprintf("sql not print by error[%v]", err))
		} else {
			tdb.logFunc(ctx, interpolateParamsSql)
		}
	}
	return err
}

func (tdb *DBFuncTemplateDB) query(db sqlDB, op *funcExecOption) error {
	if op.ctx == nil {
		op.ctx = context.Background()
	}
	rows, err := db.QueryContext(op.ctx, op.sql, op.args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	columns, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	dest, more, arrayLen, err := newScanDestByValues(tdb.sqlParamType, columns, op.result)
	if err != nil {
		return err
	}
	i := 0
	for rows.Next() {
		nextNewScanDest(op.result, dest)
		err = rows.Scan(dest...)
		if err != nil {
			return err
		}
		nextSetResult(op.result, i, dest)
		if more {
			i++
			if arrayLen > 0 && i == arrayLen {
				break
			}
		} else {
			break
		}
	}
	return nil
}

func (tdb *DBFuncTemplateDB) exec(db sqlDB, op *funcExecOption) (ret *Result, err error) {
	if op.ctx == nil {
		op.ctx = context.Background()
	}
	result, err := db.ExecContext(op.ctx, op.sql, op.args...)
	if err != nil {
		return nil, err
	}
	lastInsertId, err := result.LastInsertId()
	if err != nil {
		return &Result{}, nil
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return &Result{}, nil
	}
	return &Result{lastInsertId, rowsAffected}, nil
}

type recoverPanic struct{}

func (tdb *DBFuncTemplateDB) enableRecover(ctx context.Context) {
	if ctx != nil {
		recoverPanic, ok := ctx.Value(recoverPanic{}).(*bool)
		if ok {
			*recoverPanic = true
		}
	}
}

func (tdb *DBFuncTemplateDB) FromRecover(ctx context.Context) (*bool, bool) {
	if ctx == nil {
		return nil, false
	}
	recoverPanic, ok := ctx.Value(recoverPanic{}).(*bool)
	return recoverPanic, ok
}

func (tdb *DBFuncTemplateDB) NewRecover(ctx context.Context) context.Context {
	if _, ok := tdb.FromRecover(ctx); ok {
		return ctx
	}
	isRecoverPanic := false
	return context.WithValue(ctx, recoverPanic{}, &isRecoverPanic)
}

func (tdb *DBFuncTemplateDB) Begin(ctx context.Context) (context.Context, error) {
	return tdb.BeginTx(ctx, nil)
}

func (tdb *DBFuncTemplateDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (context.Context, error) {
	if _, ok := tdb.FromRecover(ctx); !ok {
		ctx = tdb.NewRecover(ctx)
	}
	if tx, ok := FromSqlTx(ctx); ok && tx != nil {
		return ctx, nil
	}
	tx, err := tdb.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return NewSqlTx(ctx, tx), nil
}
func (tdb *DBFuncTemplateDB) AutoCommit(ctx context.Context, err *error) {
	if err == nil {
		panic(errors.New("AutoCommit in(1) err pointer is nil"))
	}
	if *err == nil {
		if rp, ok := tdb.FromRecover(ctx); ok && *rp {
			if e := recover(); e != nil {
				switch e := e.(type) {
				case error:
					*err = e
				default:
					panic(e)
				}
			}
		}
	}
	tx, ok := FromSqlTx(ctx)
	if ok {
		if *err != nil {
			tx.Rollback()
		} else {
			*err = tx.Commit()
		}
	}
}

func (tdb *DBFuncTemplateDB) Recover(ctx context.Context, err *error) {
	if err == nil {
		panic(errors.New("Recover in(1) err pointer is nil"))
	}
	if *err == nil {
		if rp, ok := tdb.FromRecover(ctx); ok && *rp {
			if e := recover(); e != nil {
				switch e := e.(type) {
				case error:
					*err = e
				default:
					panic(e)
				}
			}
		}
	}
}

func (tdb *DBFuncTemplateDB) ParseSql(tsql string) (*template.Template, error) {
	return template.New("").Delims(tdb.leftDelim, tdb.rightDelim).SqlParams(tdb.sqlParamsConvert).
		Funcs(tdb.sqlFunc).Parse(tsql)
}

func (tdb *DBFuncTemplateDB) sqlTemplateBuild(ctx context.Context, tsql string, parms any) (string, []any, error) {
	pc, _, line, _ := runtime.Caller(2)
	if tdb.template == nil {
		tdb.template = make(map[uintptr]map[int]*template.Template)
	}
	if _, ok := tdb.template[pc]; !ok {
		tdb.template[pc] = make(map[int]*template.Template)
	}
	if _, ok := tdb.template[pc][line]; !ok {
		templateSql, err := template.New("").Delims(tdb.leftDelim, tdb.rightDelim).SqlParams(tdb.sqlParamsConvert).
			Funcs(tdb.sqlFunc).Parse(tsql)
		if err != nil {
			return "", nil, err
		}
		tdb.template[pc][line] = templateSql
	}
	templateSql := tdb.template[pc][line]
	sql, args, err := templateSql.ExecuteBuilder(parms, nil, nil)
	if err != nil {
		return "", nil, err
	}
	if tdb.sqlDebug && tdb.logFunc != nil {
		interpolateParamsSql, err := SqlInterpolateParams(sql, args)
		ctx := context.WithValue(ctx, keyLogSqlFuncName{}, fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), line))
		if err != nil {
			tdb.logFunc(ctx, fmt.Sprintf("sql not print by error[%v]", err))
		} else {
			tdb.logFunc(ctx, interpolateParamsSql)
		}
	}
	return sql, args, err
}
