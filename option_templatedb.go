package templatedb

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/tianxinzizhen/templatedb/load"
	commentStruct "github.com/tianxinzizhen/templatedb/load/comment/cstruct"
	"github.com/tianxinzizhen/templatedb/load/xml"
	"github.com/tianxinzizhen/templatedb/template"
)

// 这个版本需要进行测试,暂时先搁置比较好
// 特别是多参数时的可选参数消息
type ExecOption struct {
	Ctx      context.Context
	Sql      string
	FuncPC   uintptr
	FuncName string
	Name     string
	Param    any
	Args     []any
	Result   any
}

func FuncPC(funcP any) uintptr {
	if reflect.TypeOf(funcP).Kind() == reflect.Func {
		return reflect.ValueOf(funcP).Pointer()
	}
	return 0
}

func Args(args ...any) []any {
	return args
}

func NewExecOption() *ExecOption {
	return &ExecOption{}
}

func (op *ExecOption) SetContext(ctx context.Context) *ExecOption {
	op.Ctx = ctx
	return op
}

func (op *ExecOption) SetSql(sql string) *ExecOption {
	op.Sql = sql
	return op
}

func (op *ExecOption) SetFuncPC(funcP any) *ExecOption {
	if reflect.TypeOf(funcP).Kind() == reflect.Func {
		op.FuncPC = reflect.ValueOf(funcP).Pointer()
	}
	return op
}

func (op *ExecOption) SetFuncName(funcName string) *ExecOption {
	op.FuncName = funcName
	return op
}

func (op *ExecOption) SetName(name string) *ExecOption {
	op.Name = name
	return op
}

func (op *ExecOption) SetParam(param any) *ExecOption {
	op.Param = param
	return op
}

func (op *ExecOption) SetArgs(args ...any) *ExecOption {
	op.Args = args
	return op
}

func (op *ExecOption) SetResult(result any) *ExecOption {
	op.Result = result
	return op
}

type OptionDB struct {
	sqlDB                   *sql.DB
	template                map[string]*template.Template
	delimsLeft, delimsRight string
	sqlParamsConvert        func(val reflect.Value) any
}

type optionActionDB interface {
	query(sdb sqlDB, op *ExecOption) any
	exec(sdb sqlDB, op *ExecOption) (lastInsertId, rowsAffected int64)
}
type TemplateOptionDB interface {
	Query(op *ExecOption) any

	Exec(op *ExecOption) (lastInsertId, rowsAffected int64)

	Prepare(query string) (*sql.Stmt, error)

	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

func (db *OptionDB) Delims(delimsLeft, delimsRight string) {
	db.delimsLeft = delimsLeft
	db.delimsRight = delimsRight
}

func (db *OptionDB) SqlParamsConvert(sqlParamsConvert func(val reflect.Value) any) {
	db.sqlParamsConvert = sqlParamsConvert
}

func NewOptionDB(sqlDB *sql.DB) *OptionDB {
	db := &OptionDB{
		sqlDB:      sqlDB,
		template:   make(map[string]*template.Template),
		delimsLeft: "{", delimsRight: "}",
	}
	return db
}

func (db *OptionDB) LoadSqlOfXml(sqlfs embed.FS) error {
	if db.template == nil {
		db.template = make(map[string]*template.Template)
	}
	return xml.LoadTemplateStatements(sqlfs, db.template, db.parse)
}

func (db *OptionDB) LoadSqlOfCommentStruct(pkg string, sqlfs embed.FS) error {
	if db.template == nil {
		db.template = make(map[string]*template.Template)
	}
	return commentStruct.LoadTemplateStatements(pkg, sqlfs, db.template, db.parse)
}

func (db *OptionDB) Recover(errp *error) {
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
		recoverPrintf(*errp)
	}
}

func (db *OptionDB) parse(parse string, addParseTrees ...load.AddParseTree) (*template.Template, error) {
	templateSql, err := template.New("").Delims(db.delimsLeft, db.delimsRight).SqlParams(db.sqlParamsConvert).Funcs(sqlFunc).Parse(parse)
	if err != nil {
		return nil, err
	}
	for _, addParseTree := range addParseTrees {
		err = addParseTree(templateSql)
		if err != nil {
			return nil, err
		}
	}
	return templateSql, nil
}

func (db *OptionDB) templateBuild(opSql string, opFuncPc uintptr, opFuncName string, opName string, opParams any, opArgs []any) (string, []any, error) {
	if db.template == nil {
		db.template = make(map[string]*template.Template)
	}
	var line int
	if opFuncName == "" {
		if opFuncPc == 0 {
			opFuncPc, _, line, _ = runtime.Caller(3)
		}
		opFuncName = runtime.FuncForPC(opFuncPc).Name()
	}
	tKey := fmt.Sprintf("%s:%s", opFuncName, opName)
	templateSql, templateok := db.template[tKey]
	if !templateok {
		tKey = fmt.Sprintf("%s:%d", opFuncName, line)
		templateSql, templateok = db.template[tKey]
		if !templateok {
			if len(strings.Trim(opSql, "\t\n\f\r ")) == 0 {
				return "", nil, fmt.Errorf("template sql string is empy")
			}
			var err error
			templateSql, err = db.parse(opSql)
			if err != nil {
				return "", nil, err
			}
			db.template[tKey] = templateSql
		}
	}
	sql, args, err := templateSql.ExecuteBuilder(opParams, opArgs)
	return sql, args, err
}

func (db *OptionDB) query(sdb sqlDB, op *ExecOption) any {
	sql, args, err := db.templateBuild(op.Sql, op.FuncPC, op.FuncName, op.Name, op.Param, op.Args)
	if err != nil {
		panic(err)
	}
	if op.Ctx == nil {
		op.Ctx = context.Background()
	}
	rows, err := sdb.QueryContext(op.Ctx, sql, args...)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	columns, err := rows.ColumnTypes()
	if err != nil {
		panic(err)
	}
	rt := reflect.TypeOf(op.Result)
	rv := reflect.ValueOf(op.Result)
	if rt.Kind() == reflect.Func {
		if rt.NumIn() == 1 {
			ft := rt.In(0)
			dest := newScanDest(columns, ft)
			for rows.Next() {
				receiver := newReceiver(rt, columns, dest)
				err = rows.Scan(dest...)
				if err != nil {
					panic(err)
				}
				rv.Call([]reflect.Value{receiver})
			}
		} else {
			dest := newScanDest(columns, rt)
			for rows.Next() {
				receiver := newReceiver(rt, columns, dest)
				err = rows.Scan(dest...)
				if err != nil {
					panic(err)
				}
				rv.Call(receiver.Interface().([]reflect.Value))
			}
		}
		return nil
	} else {
		st := rt
		if rt.Kind() == reflect.Slice {
			if rv.IsNil() {
				rv = reflect.MakeSlice(rt, 0, 10)
			}
			st = rt.Elem()
		} else {
			if rt.Kind() == reflect.Pointer && rv.IsNil() {
				rv = reflect.New(rt).Elem()
			}
		}
		dest := newScanDest(columns, st)
		for rows.Next() {
			receiver := newReceiver(st, columns, dest)
			err = rows.Scan(dest...)
			if err != nil {
				panic(err)
			}
			if rt.Kind() == reflect.Slice {
				rv = reflect.Append(rv, receiver)
			} else {
				return receiver.Interface()
			}
		}
		return rv.Interface()
	}
}

func (db *OptionDB) exec(sdb sqlDB, op *ExecOption) (lastInsertId, rowsAffected int64) {
	sql, args, err := db.templateBuild(op.Sql, op.FuncPC, op.FuncName, op.Name, op.Param, op.Args)
	if err != nil {
		panic(err)
	}
	if op.Ctx == nil {
		op.Ctx = context.Background()
	}
	result, err := sdb.ExecContext(op.Ctx, sql, args...)
	if err != nil {
		panic(err)
	}
	lastid, err := result.LastInsertId()
	if err != nil {
		panic(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		panic(err)
	}
	return lastid, affected
}
func (db *OptionDB) Query(op *ExecOption) any {
	return db.query(db.sqlDB, op)
}
func (db *OptionDB) Exec(op *ExecOption) (lastInsertId, rowsAffected int64) {
	return db.exec(db.sqlDB, op)
}
func (db *OptionDB) Prepare(query string) (*sql.Stmt, error) {
	return db.sqlDB.Prepare(query)
}

func (db *OptionDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return db.sqlDB.PrepareContext(ctx, query)
}

func (db *OptionDB) Begin() (*OptionTxDB, error) {
	return db.BeginTx(context.Background(), nil)
}

func (db *OptionDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*OptionTxDB, error) {
	tx, err := db.sqlDB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &OptionTxDB{optionActionDB: db, tx: tx}, nil
}

type OptionTxDB struct {
	optionActionDB
	tx *sql.Tx
}

func (tx *OptionTxDB) AutoCommit(errp *error) {
	if *errp == nil {
		e := recover()
		if e != nil {
			switch err := e.(type) {
			case error:
				*errp = err
			default:
				panic(e)
			}
			recoverPrintf(*errp)
		}
	}
	if *errp != nil {
		tx.tx.Rollback()
	} else {
		tx.tx.Commit()
	}
}

func (db *OptionTxDB) Query(op *ExecOption) any {
	return db.query(db.tx, op)
}

func (db *OptionTxDB) Exec(op *ExecOption) (lastInsertId, rowsAffected int64) {
	return db.exec(db.tx, op)
}

func (db *OptionTxDB) Prepare(query string) (*sql.Stmt, error) {
	return db.tx.Prepare(query)
}

func (db *OptionTxDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return db.tx.PrepareContext(ctx, query)
}
