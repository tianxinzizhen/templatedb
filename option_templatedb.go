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
	query(sdb sqlDB, op *ExecOption) (any, error)
	exec(sdb sqlDB, op *ExecOption) (lastInsertId, rowsAffected int64, err error)
}
type TemplateOptionDB interface {
	Query(op *ExecOption) (any, error)
	TQuery(op *ExecOption) any

	Exec(op *ExecOption) (lastInsertId, rowsAffected int64, err error)
	TExec(op *ExecOption) (lastInsertId, rowsAffected int64)

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

func (db *OptionDB) query(sdb sqlDB, op *ExecOption) (any, error) {
	sql, args, err := db.templateBuild(op.Sql, op.FuncPC, op.FuncName, op.Name, op.Param, op.Args)
	if err != nil {
		return nil, err
	}
	if op.Ctx == nil {
		op.Ctx = context.Background()
	}
	rows, err := sdb.QueryContext(op.Ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	rt := reflect.TypeOf(op.Result)
	rv := reflect.ValueOf(op.Result)
	if rt.Kind() == reflect.Func {
		if rt.NumIn() == 1 {
			ft := rt.In(0)
			dest, err := newScanDest(columns, ft)
			if err != nil {
				return nil, err
			}
			for rows.Next() {
				receiver := newReceiver(rt, columns, dest)
				err = rows.Scan(dest...)
				if err != nil {
					return nil, err
				}
				rv.Call([]reflect.Value{receiver})
			}
		} else {
			dest, err := newScanDest(columns, rt)
			if err != nil {
				return nil, err
			}
			for rows.Next() {
				receiver := newReceiver(rt, columns, dest)
				err = rows.Scan(dest...)
				if err != nil {
					return nil, err
				}
				if rt.NumIn() > 0 {
					rv.Call(receiver.Interface().([]reflect.Value))
				} else {
					return receiver.Interface(), nil
				}

			}
		}
		return nil, nil
	} else {
		for rt.Kind() == reflect.Pointer {
			if rv.IsNil() {
				var tv reflect.Value
				if rt.Elem().Kind() == reflect.Slice {
					tv = reflect.NewAt(rt.Elem(), reflect.MakeSlice(rt.Elem(), 0, 10).UnsafePointer())
				} else {
					tv = reflect.New(rt).Elem()
				}
				if rv.CanSet() {
					rv.Set(tv)
				} else {
					rv = tv
				}
			}
			rt = rt.Elem()
			rv = rv.Elem()
			if rv.Kind() == 0 {
				break
			}
		}
		st := rt
		if rt.Kind() == reflect.Slice {
			if rv.IsNil() {
				if rv.CanSet() {
					rv.Set(reflect.MakeSlice(rt, 0, 10))
				} else {
					rv = reflect.NewAt(rt, reflect.MakeSlice(rt, 0, 10).UnsafePointer()).Elem()
				}
			}
			if !rv.CanSet() {
				rv = reflect.NewAt(rt, rv.UnsafePointer()).Elem()
			}
			rt = rt.Elem()
		}
		dest, err := newScanDest(columns, rt)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			row := newReceiver(rt, columns, dest)
			err = rows.Scan(dest...)
			if err != nil {
				return nil, err
			}
			if rv.CanSet() {
				if st.Kind() == reflect.Slice {
					rv.Set(reflect.Append(rv, row))
				} else {
					rv.Set(row)
					break
				}
			} else {
				if st.Kind() == reflect.Slice {
					rv = reflect.Append(rv, row)
				} else {
					rv = row
					break
				}
			}

		}
		for reflect.PtrTo(rv.Type()) != reflect.PtrTo(reflect.TypeOf(op.Result)) {
			if rv.CanAddr() {
				rv = rv.Addr()
			} else {
				break
			}
		}
		return rv.Interface(), nil
	}
}

func (db *OptionDB) exec(sdb sqlDB, op *ExecOption) (lastInsertId, rowsAffected int64, err error) {
	sql, args, err := db.templateBuild(op.Sql, op.FuncPC, op.FuncName, op.Name, op.Param, op.Args)
	if err != nil {
		return 0, 0, err
	}
	if op.Ctx == nil {
		op.Ctx = context.Background()
	}
	result, err := sdb.ExecContext(op.Ctx, sql, args...)
	if err != nil {
		return 0, 0, err
	}
	lastid, err := result.LastInsertId()
	if err != nil {
		return 0, 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, 0, err
	}
	return lastid, affected, nil
}
func (db *OptionDB) Query(op *ExecOption) (any, error) {
	return db.query(db.sqlDB, op)
}
func (db *OptionDB) Exec(op *ExecOption) (lastInsertId, rowsAffected int64, err error) {
	return db.exec(db.sqlDB, op)
}
func (db *OptionDB) TQuery(op *ExecOption) any {
	rows, err := db.query(db.sqlDB, op)
	if err != nil {
		panic(err)
	}
	return rows
}
func (db *OptionDB) TExec(op *ExecOption) (lastInsertId, rowsAffected int64) {
	lastInsertId, rowsAffected, err := db.exec(db.sqlDB, op)
	if err != nil {
		panic(err)
	}
	return
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

func (db *OptionTxDB) Query(op *ExecOption) (any, error) {
	return db.query(db.tx, op)
}
func (db *OptionTxDB) Exec(op *ExecOption) (lastInsertId, rowsAffected int64, err error) {
	return db.exec(db.tx, op)
}
func (db *OptionTxDB) TQuery(op *ExecOption) any {
	rows, err := db.query(db.tx, op)
	if err != nil {
		panic(err)
	}
	return rows
}
func (db *OptionTxDB) TExec(op *ExecOption) (lastInsertId, rowsAffected int64) {
	lastInsertId, rowsAffected, err := db.exec(db.tx, op)
	if err != nil {
		panic(err)
	}
	return
}

func (db *OptionTxDB) Prepare(query string) (*sql.Stmt, error) {
	return db.tx.Prepare(query)
}

func (db *OptionTxDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return db.tx.PrepareContext(ctx, query)
}