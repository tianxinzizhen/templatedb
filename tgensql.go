package tgsql

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"reflect"
	"runtime"

	"github.com/tianxinzizhen/tgsql/load"
	"github.com/tianxinzizhen/tgsql/sqlval"
	"github.com/tianxinzizhen/tgsql/sqlwrite"
	"github.com/tianxinzizhen/tgsql/template"
	"github.com/tianxinzizhen/tgsql/util"
)

type TgenSql struct {
	db                      *sql.DB
	localFuncDataInfo       *load.LoadFuncDataInfo
	leftDelim, rightDelim   string
	sqlLogFunc              func(ctx context.Context, funcName, sql string, args ...any)
	filedName               template.FiledName
	sqlFunc                 template.FuncMap
	template                map[uintptr]map[int]*template.Template
	SqlEscapeBytesBackslash bool
}

func (tdb *TgenSql) SetSqlEscapeBytesBackslash(sqlEscapeBytesBackslash bool) {
	tdb.SqlEscapeBytesBackslash = sqlEscapeBytesBackslash
}

func (tdb *TgenSql) Delims(leftDelim, rightDelim string) {
	tdb.leftDelim = leftDelim
	tdb.rightDelim = rightDelim
}

func (tdb *TgenSql) SqlLogFunc(sqlLogFunc func(ctx context.Context, funcName, sql string, args ...any)) {
	tdb.sqlLogFunc = sqlLogFunc
}

func (tdb *TgenSql) AddTemplateFunc(key string, funcMethod any) {
	tdb.sqlFunc[key] = funcMethod
}

func (tdb *TgenSql) AddAllTemplateFunc(sqlFunc template.FuncMap) {
	for k, v := range sqlFunc {
		tdb.sqlFunc[k] = v
	}
}

func (tdb *TgenSql) LoadFuncDataInfo(dbFuncData embed.FS) error {
	return tdb.localFuncDataInfo.LoadFuncDataInfo(dbFuncData)
}

func (tdb *TgenSql) LoadFuncDataInfoBytes(dbFuncData []byte) error {
	return tdb.localFuncDataInfo.LoadFuncDataInfoBytes(dbFuncData)
}

func (tdb *TgenSql) LoadFuncDataInfoString(dbFuncData string) error {
	return tdb.localFuncDataInfo.LoadFuncDataInfoString(dbFuncData)
}

func NewTgenSql(sqlDB *sql.DB) *TgenSql {
	tdb := &TgenSql{
		db:        sqlDB,
		leftDelim: "{", rightDelim: "}",
		sqlFunc:           make(template.FuncMap),
		filedName:         template.DefaultFieldName,
		localFuncDataInfo: load.NewLoadFuncDataInfo(),
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

func (tdb *TgenSql) templateBuild(templateSql *template.Template, op *funcExecOption) error {
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
func (tdb *TgenSql) query(op *funcExecOption) error {
	return tdb.queryOption(op, queryOption{})
}

type queryOption struct {
	selectOne bool
}

func (tdb *TgenSql) queryOption(op *funcExecOption, queryOption queryOption) error {
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

func (tdb *TgenSql) exec(op *funcExecOption) (ret sql.Result, err error) {
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

func (tdb *TgenSql) prepareContext(op *funcExecOption) (ret *sql.Stmt, err error) {
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

func (tdb *TgenSql) enableRecover(ctx context.Context) {
	if ctx != nil {
		recoverPanic, ok := ctx.Value(recoverPanic{}).(*bool)
		if ok {
			*recoverPanic = true
		}
	}
}

func (tdb *TgenSql) FromRecover(ctx context.Context) (*bool, bool) {
	if ctx == nil {
		return nil, false
	}
	recoverPanic, ok := ctx.Value(recoverPanic{}).(*bool)
	return recoverPanic, ok
}

func (tdb *TgenSql) ParseSql(tsql string) (*template.Template, error) {
	return template.New("").Delims(tdb.leftDelim, tdb.rightDelim).
		SetFieldName(tdb.filedName).
		Funcs(tdb.sqlFunc).Parse(tsql)
}

func (tdb *TgenSql) sqlTemplateBuild(ctx context.Context, tsql string, parms any) (string, []any, error) {
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

func (tdb *TgenSql) sqlPrintAndRecord(ctx context.Context, funcName, sql string, args []any) {
	if tdb.sqlLogFunc == nil {
		return
	}
	tdb.sqlLogFunc(ctx, funcName, sql, args...)
	if recordSql, ok := tdb.FromRecordSql(ctx); ok {
		recordSql.List = append(recordSql.List, RecordSqlItem{Sql: sql, Args: args})
	}
}
