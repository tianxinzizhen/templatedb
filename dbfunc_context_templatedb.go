package templatedb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"runtime"

	"github.com/tianxinzizhen/templatedb/sqlval"
	"github.com/tianxinzizhen/templatedb/sqlwrite"
	"github.com/tianxinzizhen/templatedb/template"
	"github.com/tianxinzizhen/templatedb/util"
)

type DBFuncTemplateDB struct {
	db                      *sql.DB
	leftDelim, rightDelim   string
	sqlDebug                bool
	logFunc                 func(ctx context.Context, info string)
	filedName               template.FiledName
	sqlFunc                 template.FuncMap
	template                map[uintptr]map[int]*template.Template
	SqlEscapeBytesBackslash bool
}

func (tdb *DBFuncTemplateDB) SetSqlEscapeBytesBackslash(sqlEscapeBytesBackslash bool) {
	tdb.SqlEscapeBytesBackslash = sqlEscapeBytesBackslash
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

func NewDBFuncTemplateDB(sqlDB *sql.DB) *DBFuncTemplateDB {
	tdb := &DBFuncTemplateDB{
		db:        sqlDB,
		leftDelim: "{", rightDelim: "}",
		sqlFunc:   make(template.FuncMap),
		filedName: template.DefaultFieldName,
	}
	for k, v := range sqlFunc {
		tdb.sqlFunc[k] = v
	}
	return tdb
}

const (
	OptionNone int = 1 << iota
	OptionNotPrepare
	OptionBatchInsert
)

type funcExecOption struct {
	ctx    context.Context
	param  any
	result []reflect.Value
	sql    string
	args   []any
	option int
	offset int
	db     any
	stmt   *sql.Stmt
}

func (op *funcExecOption) GetDB(ctx context.Context) any {
	if op.stmt != nil {
		return op.stmt
	}
	tx, ok := FromSqlTx(ctx)
	if ok && tx != nil {
		return tx
	}
	return op.db
}

type keyLogSqlFuncName struct{}

func FromLogSqlFuncName(ctx context.Context) (sql string, ok bool) {
	sql, ok = ctx.Value(keyLogSqlFuncName{}).(string)
	return
}

func (tdb *DBFuncTemplateDB) templateBuild(templateSql *template.Template, op *funcExecOption) error {
	sqlWrite := &sqlwrite.SqlWrite{}
	err := templateSql.Execute(sqlWrite, op.param)
	if err != nil {
		return err
	}
	if op.option&OptionNotPrepare != 0 {
		op.sql, err = util.InterpolateParams(sqlWrite.Sql(), sqlWrite.Args(), tdb.SqlEscapeBytesBackslash)
		if err != nil {
			return err
		}
		op.args = nil
	} else {
		op.sql = sqlWrite.Sql()
		op.args = sqlWrite.Args()
	}
	tdb.sqlPrintAndRecord(op.ctx, templateSql.Name(), op.sql, op.args)
	return err
}
func (tdb *DBFuncTemplateDB) query(op *funcExecOption) error {
	return tdb.queryOption(op, queryOption{})
}

type queryOption struct {
	selectOne bool
}

func (tdb *DBFuncTemplateDB) queryOption(op *funcExecOption, queryOption queryOption) error {
	if op.ctx == nil {
		op.ctx = context.Background()
	}
	db := op.GetDB(op.ctx).(sqlDB)
	var err error
	op.args, err = sqlval.ConvertValues(op.db, op.args)
	if err != nil {
		return err
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
	for rows.Next() {
		dest, df, err := sqlval.GetScanDest(tdb.filedName, columns, op.result)
		if err != nil {
			return err
		}
		err = rows.Scan(dest...)
		if err != nil {
			return err
		}
		for _, fn := range df {
			fn()
		}
		if queryOption.selectOne {
			break
		}
	}
	return nil
}

func (tdb *DBFuncTemplateDB) exec(op *funcExecOption) (ret sql.Result, err error) {
	if op.ctx == nil {
		op.ctx = context.Background()
	}
	switch db := op.GetDB(op.ctx).(type) {
	case sqlDB:
		op.args, err = sqlval.ConvertValues(op.db, op.args)
		if err != nil {
			return nil, err
		}
		result, err := db.ExecContext(op.ctx, op.sql, op.args...)
		if err != nil {
			return nil, err
		}
		return result, nil
	case sqlStmt:
		result, err := db.ExecContext(op.ctx, op.args...)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return nil, errors.New("db not support exec")
}

func (tdb *DBFuncTemplateDB) prepareContext(op *funcExecOption) (ret *sql.Stmt, err error) {
	if op.ctx == nil {
		op.ctx = context.Background()
	}
	if db, ok := op.GetDB(op.ctx).(sqlPrepare); ok {
		result, err := db.PrepareContext(op.ctx, op.sql)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return nil, errors.New("db not support prepare")
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
		SetFieldName(tdb.filedName).
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
	sqw := &sqlwrite.SqlWrite{}
	err := templateSql.Execute(sqw, parms)
	if err != nil {
		return "", nil, err
	}
	tdb.sqlPrintAndRecord(ctx, fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), line), sqw.Sql(), sqw.Args())
	return sqw.Sql(), sqw.Args(), err
}

func (tdb *DBFuncTemplateDB) sqlPrintAndRecord(ctx context.Context, sqlFuncName, sql string, args []any) {
	needPrintSql := (tdb.sqlDebug && tdb.logFunc != nil)
	recordSql, recordSqlOk := tdb.FromRecordSql(ctx)
	if needPrintSql || recordSqlOk {
		interpolateParamsSql, err := util.InterpolateParams(sql, args, tdb.SqlEscapeBytesBackslash)
		if needPrintSql {
			ctx = context.WithValue(ctx, keyLogSqlFuncName{}, sqlFuncName)
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
