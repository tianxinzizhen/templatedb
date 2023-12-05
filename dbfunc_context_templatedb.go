package templatedb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"time"

	"github.com/tianxinzizhen/templatedb/template"
)

type DBFuncTemplateDB struct {
	db                    *sql.DB
	leftDelim, rightDelim string
	sqlDebug              bool
	logFunc               func(ctx context.Context, info string)
	getFieldByName        func(t reflect.Type, fieldName string, scanNum map[string]int) (f reflect.StructField, ok bool)
	getParameterMap       map[reflect.Type]func(any) (string, any, error)
	setParameterMap       map[reflect.Type]func(src any) (any, error)
	sqlFunc               template.FuncMap
	template              map[uintptr]map[int]*template.Template
}

func (tdb *DBFuncTemplateDB) Delims(leftDelim, rightDelim string) {
	tdb.leftDelim = leftDelim
	tdb.rightDelim = rightDelim
}

func (tdb *DBFuncTemplateDB) SqlDebug(sqlDebug bool) {
	tdb.sqlDebug = sqlDebug
}

func (tdb *DBFuncTemplateDB) LogFunc(logFunc func(ctx context.Context, info string)) {
	tdb.logFunc = logFunc
}

func (tdb *DBFuncTemplateDB) AddTemplateFunc(key string, funcMethod any) {
	tdb.sqlFunc[key] = funcMethod
}

func (tdb *DBFuncTemplateDB) AddAllTemplateFunc(sqlFunc template.FuncMap) {
	for k, v := range sqlFunc {
		tdb.sqlFunc[k] = v
	}
}

func (tdb *DBFuncTemplateDB) AddGetParameter(t reflect.Type, getParameter func(any) (string, any, error)) error {
	if _, ok := tdb.getParameterMap[t]; ok {
		return fmt.Errorf("add GetParameter type [%s] already exists ", t)
	} else {
		tdb.getParameterMap[t] = getParameter
	}
	return nil
}

func (tdb *DBFuncTemplateDB) AddSetParameter(t reflect.Type, setParameter func(src any) (any, error)) error {
	if _, ok := tdb.setParameterMap[t]; ok {
		return fmt.Errorf("add SetParameter type [%s] already exists ", t)
	} else {
		tdb.setParameterMap[t] = setParameter
	}
	return nil
}

func (tdb *DBFuncTemplateDB) AddScan(t reflect.Type, setParameter func(src any) (any, error)) error {
	return tdb.AddSetParameter(t, setParameter)
}

func (tdb *DBFuncTemplateDB) GetFieldByName(getFieldByName func(t reflect.Type, fieldName string, scanNum map[string]int) (f reflect.StructField, ok bool)) {
	if getFieldByName != nil {
		tdb.getFieldByName = getFieldByName
	}
}

func NewDBFuncTemplateDB(sqlDB *sql.DB) *DBFuncTemplateDB {
	tdb := &DBFuncTemplateDB{
		db:        sqlDB,
		leftDelim: "{", rightDelim: "}",
		sqlFunc:         make(template.FuncMap),
		getFieldByName:  DefaultGetFieldByName,
		getParameterMap: make(map[reflect.Type]func(any) (string, any, error)),
		setParameterMap: make(map[reflect.Type]func(src any) (any, error)),
	}
	//default time get paramter
	tp := reflect.TypeOf(&time.Time{})
	tp_get := func(t any) (string, any, error) {
		return "?", t, nil
	}
	tdb.getParameterMap[tp] = tp_get
	tdb.getParameterMap[tp.Elem()] = tp_get
	for k, v := range sqlFunc {
		tdb.sqlFunc[k] = v
	}
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
	tdb.sqlPrintAndRecord(op.ctx, templateSql.Name(), op.sql, op.args)
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
	dest, more, arrayLen, err := tdb.newScanDestByValues(columns, op.result)
	if err != nil {
		return err
	}
	i := 0
	for rows.Next() {
		tdb.nextNewScanDest(op.result, dest)
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

type recordSqlKey struct{}
type RecordSql struct {
	Sql []string
}

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

func (tdb *DBFuncTemplateDB) FromRecordSql(ctx context.Context) (*RecordSql, bool) {
	if ctx == nil {
		return nil, false
	}
	recordSql, ok := ctx.Value(recordSqlKey{}).(*RecordSql)
	return recordSql, ok
}

func (tdb *DBFuncTemplateDB) NewRecover(ctx context.Context) context.Context {
	if _, ok := tdb.FromRecover(ctx); ok {
		return ctx
	}
	isRecoverPanic := false
	return context.WithValue(ctx, recoverPanic{}, &isRecoverPanic)
}

func (tdb *DBFuncTemplateDB) NewRecordSql(ctx context.Context) context.Context {
	if _, ok := tdb.FromRecordSql(ctx); ok {
		return ctx
	}
	return context.WithValue(ctx, recordSqlKey{}, &RecordSql{})
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

func (tdb *DBFuncTemplateDB) Rollback(ctx context.Context) error {
	tx, ok := FromSqlTx(ctx)
	if ok {
		return tx.Rollback()
	}
	return nil
}

func (tdb *DBFuncTemplateDB) Commit(ctx context.Context) error {
	tx, ok := FromSqlTx(ctx)
	if ok {
		return tx.Commit()
	}
	return nil
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
	return template.New("").Delims(tdb.leftDelim, tdb.rightDelim).
		GetFieldByName(tdb.getFieldByName).
		GetParameterMap(tdb.getParameterMap).
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
		templateSql, err := tdb.ParseSql(tsql)
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
	tdb.sqlPrintAndRecord(ctx, fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), line), sql, args)
	return sql, args, err
}

func (tdb *DBFuncTemplateDB) sqlPrintAndRecord(ctx context.Context, sqlFuncName, sql string, args []any) {
	needPrintSql := (tdb.sqlDebug && tdb.logFunc != nil)
	recordSql, recordSqlOk := tdb.FromRecordSql(ctx)
	if needPrintSql || recordSqlOk {
		interpolateParamsSql, err := SqlInterpolateParams(sql, args)
		if needPrintSql {
			ctx := context.WithValue(ctx, keyLogSqlFuncName{}, sqlFuncName)
			if err != nil {
				tdb.logFunc(ctx, fmt.Sprintf("sql not print by error[%v]", err))
			} else {
				tdb.logFunc(ctx, interpolateParamsSql)
			}
		}
		if recordSqlOk {
			if err == nil {
				recordSql.Sql = append(recordSql.Sql, interpolateParamsSql)
			}
		}
	}
}
