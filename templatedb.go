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

type actionDB interface {
	selectScanFunc(ctx context.Context, adb sqlDB, params any, scanFunc any, name []any)
	exec(ctx context.Context, adb sqlDB, params any, name []any) (lastInsertId, rowsAffected int64)
	prepareExecContext(ctx context.Context, adb sqlDB, params []any, name []any) (rowsAffected int64)
	selectCommon(ctx context.Context, sdb sqlDB, params any, t reflect.Type, cap int, name []any) reflect.Value
}

type TemplateDB interface {
	Exec(params any, name ...any) (lastInsertId, rowsAffected int64)
	ExecContext(ctx context.Context, params any, name ...any) (lastInsertId, rowsAffected int64)
	PrepareExec(params []any, name ...any) (rowsAffected int64)
	PrepareExecContext(ctx context.Context, params []any, name ...any) (rowsAffected int64)
	SelectScanFunc(params any, scanFunc any, name ...any)
	SelectScanFuncContext(ctx context.Context, params any, scanFunc any, name ...any)
	selectByType(ctx context.Context, params any, t reflect.Type, name ...any) reflect.Value
}

type DefaultDB struct {
	sqlDB                   *sql.DB
	template                map[string]*template.Template
	delimsLeft, delimsRight string
	sqlParams               func(val reflect.Value) any
}

func getSkipFuncName(skip int, name []any) string {
	if len(name) > 0 && reflect.TypeOf(name[0]).Kind() == reflect.Func {
		return fmt.Sprintf("%s:%s", runtime.FuncForPC(reflect.ValueOf(name[0]).Pointer()).Name(), fmt.Sprint(name[1:]...))
	}
	if len(name) > 1 && reflect.TypeOf(name[0]).Kind() == reflect.String {
		return fmt.Sprintf("%s:%s", name[0], fmt.Sprint(name[1:]...))
	}
	pc, _, _, _ := runtime.Caller(skip)
	funcName := runtime.FuncForPC(pc).Name()
	return fmt.Sprintf("%s:%s", funcName, fmt.Sprint(name...))
}

func Delims(delimsLeft, delimsRight string) func(*DefaultDB) error {
	return func(db *DefaultDB) error {
		db.delimsLeft = delimsLeft
		db.delimsRight = delimsRight
		return nil
	}
}

func SqlParams(sqlParams func(val reflect.Value) any) func(*DefaultDB) error {
	return func(db *DefaultDB) error {
		db.sqlParams = sqlParams
		return nil
	}
}

func NewDefaultDB(sqlDB *sql.DB, options ...func(*DefaultDB) error) (*DefaultDB, error) {
	db := &DefaultDB{
		sqlDB:      sqlDB,
		template:   make(map[string]*template.Template),
		delimsLeft: "{", delimsRight: "}",
	}
	for _, fn := range options {
		err := fn(db)
		if err != nil {
			return nil, err
		}
	}
	return db, nil
}

func (db *DefaultDB) LoadSqlOfXml(sqlfs embed.FS) error {
	if db.template == nil {
		db.template = make(map[string]*template.Template)
	}
	return xml.LoadTemplateStatements(sqlfs, db.template, db.parse)
}

func (db *DefaultDB) LoadSqlOfCommentStruct(pkg string, sqlfs embed.FS) error {
	if db.template == nil {
		db.template = make(map[string]*template.Template)
	}
	return commentStruct.LoadTemplateStatements(pkg, sqlfs, db.template, db.parse)
}

func (db *DefaultDB) Recover(errp *error) {
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

func (db *DefaultDB) parse(parse string, addParseTrees ...load.AddParseTree) (*template.Template, error) {
	templateSql, err := template.New("").Delims(db.delimsLeft, db.delimsRight).SqlParams(db.sqlParams).Funcs(sqlFunc).Parse(parse)
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

func (db *DefaultDB) templateBuild(query string, params any) (sql string, args []any, err error) {
	if db.template == nil {
		db.template = make(map[string]*template.Template)
	}
	templateSql, templateok := db.template[query]
	if !templateok {
		_, query, _ := strings.Cut(query, ":")
		if len(strings.Trim(query, "\t\n\f\r ")) == 0 {
			return "", nil, fmt.Errorf("template sql string is empy")
		}
		templateSql, err = db.parse(query)
		if err != nil {
			return
		}
		db.template[query] = templateSql
	}
	return templateSql.ExecuteBuilder(params, nil)
}

func (db *DefaultDB) selectScanFunc(ctx context.Context, sdb sqlDB, params any, scanFunc any, name []any) {
	statement := getSkipFuncName(3, name)
	sql, args, err := db.templateBuild(statement, params)
	if err != nil {
		panic(err)
	}
	rows, err := sdb.QueryContext(ctx, sql, args...)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	columns, err := rows.ColumnTypes()
	if err != nil {
		panic(err)
	}
	st := reflect.TypeOf(scanFunc)
	if st.Kind() != reflect.Func {
		panic(fmt.Errorf("parameter scanFunc is not function"))
	}
	if st.NumIn() == 1 {
		t := st.In(0)
		dest := newScanDest(columns, t)
		for rows.Next() {
			receiver := newReceiver(t, columns, dest)
			err = rows.Scan(dest...)
			if err != nil {
				panic(err)
			}
			reflect.ValueOf(scanFunc).Call([]reflect.Value{receiver})
		}
	} else {
		dest := newScanDest(columns, st)
		for rows.Next() {
			receiver := newReceiver(st, columns, dest)
			err = rows.Scan(dest...)
			if err != nil {
				panic(err)
			}
			reflect.ValueOf(scanFunc).Call(receiver.Interface().([]reflect.Value))
		}
	}
}

func (db *DefaultDB) selectCommon(ctx context.Context, sdb sqlDB, params any, t reflect.Type, cap int, name []any) reflect.Value {
	statement := getSkipFuncName(3, name)
	sql, args, err := db.templateBuild(statement, params)
	if err != nil {
		panic(err)
	}
	rows, err := sdb.QueryContext(ctx, sql, args...)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	columns, err := rows.ColumnTypes()
	if err != nil {
		panic(err)
	}
	var ret reflect.Value
	st := t
	if t.Kind() == reflect.Slice {
		if cap <= 0 {
			cap = 10
		}
		ret = reflect.MakeSlice(t, 0, cap)
		st = t.Elem()
	} else {
		ret = reflect.New(t).Elem()
	}
	dest := newScanDest(columns, st)
	for rows.Next() {
		receiver := newReceiver(st, columns, dest)
		err = rows.Scan(dest...)
		if err != nil {
			panic(err)
		}
		if t.Kind() == reflect.Slice {
			ret = reflect.Append(ret, receiver)
		} else {
			return receiver
		}
	}
	return ret
}

func (db *DefaultDB) exec(ctx context.Context, sdb sqlDB, params any, name []any) (lastInsertId, rowsAffected int64) {
	statement := getSkipFuncName(3, name)
	tsql, args, err := db.templateBuild(statement, params)
	if err != nil {
		panic(err)
	}
	result, err := sdb.ExecContext(ctx, tsql, args...)
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

func (db *DefaultDB) prepareExecContext(ctx context.Context, sdb sqlDB, params []any, name []any) (rowsAffected int64) {
	statement := getSkipFuncName(3, name)
	var stmtMaps map[string]*sql.Stmt = make(map[string]*sql.Stmt)
	var tempSql string
	for _, param := range params {
		execSql, args, err := db.templateBuild(statement, param)
		if err != nil {
			panic(err)
		}
		var stmt *sql.Stmt
		if tempSql != execSql {
			tempSql = execSql
			if s, ok := stmtMaps[execSql]; !ok {
				stmt, err = sdb.PrepareContext(ctx, execSql)
				if err != nil {
					panic(err)
				}
				stmtMaps[execSql] = stmt
			} else {
				stmt = s
			}
		} else {
			stmt = stmtMaps[execSql]
		}
		result, err := stmt.ExecContext(ctx, args...)
		if err != nil {
			panic(err)
		}
		batchAffected, err := result.RowsAffected()
		if err != nil {
			panic(err)
		}
		rowsAffected += batchAffected
	}
	return
}

func (db *DefaultDB) Exec(params any, name ...any) (lastInsertId, rowsAffected int64) {
	return db.exec(context.Background(), db.sqlDB, params, name)
}

func (db *DefaultDB) ExecContext(ctx context.Context, params any, name ...any) (lastInsertId, rowsAffected int64) {
	return db.exec(ctx, db.sqlDB, params, name)
}

func (db *DefaultDB) PrepareExec(params []any, name ...any) (rowsAffected int64) {
	return db.prepareExecContext(context.Background(), db.sqlDB, params, name)
}

func (db *DefaultDB) PrepareExecContext(ctx context.Context, params []any, name ...any) (rowsAffected int64) {
	return db.prepareExecContext(ctx, db.sqlDB, params, name)
}

func (db *DefaultDB) SelectScanFunc(params any, scanFunc any, name ...any) {
	db.selectScanFunc(context.Background(), db.sqlDB, params, scanFunc, name)
}

func (db *DefaultDB) SelectScanFuncContext(ctx context.Context, params any, scanFunc any, name ...any) {
	db.selectScanFunc(ctx, db.sqlDB, params, scanFunc, name)
}

func (db *DefaultDB) selectByType(ctx context.Context, params any, t reflect.Type, name ...any) reflect.Value {
	return db.selectCommon(ctx, db.sqlDB, params, t, 0, name)
}

func (db *DefaultDB) RawExec(query string, args ...any) (sql.Result, error) {
	return db.sqlDB.Exec(query, args...)
}

func (db *DefaultDB) RawExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.sqlDB.ExecContext(ctx, query, args...)
}

func (db *DefaultDB) RawQuery(query string, args ...any) (*sql.Rows, error) {
	return db.sqlDB.Query(query, args...)
}

func (db *DefaultDB) RawQueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.sqlDB.QueryContext(ctx, query, args...)
}

func (db *DefaultDB) Begin() (*TemplateTxDB, error) {
	return db.BeginTx(context.Background(), nil)
}

func (db *DefaultDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*TemplateTxDB, error) {
	tx, err := db.sqlDB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &TemplateTxDB{actionDB: db, tx: tx}, nil
}

type TemplateTxDB struct {
	actionDB
	tx *sql.Tx
}

func (tx *TemplateTxDB) AutoCommit(errp *error) {
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

func (tx *TemplateTxDB) Exec(params any, name ...any) (lastInsertId, rowsAffected int64) {
	return tx.exec(context.Background(), tx.tx, params, name)
}

func (tx *TemplateTxDB) ExecContext(ctx context.Context, params any, name ...any) (lastInsertId, rowsAffected int64) {
	return tx.exec(ctx, tx.tx, params, name)
}

func (tx *TemplateTxDB) PrepareExec(params []any, name ...any) (rowsAffected int64) {
	return tx.prepareExecContext(context.Background(), tx.tx, params, name)
}

func (tx *TemplateTxDB) PrepareExecContext(ctx context.Context, params []any, name ...any) (rowsAffected int64) {
	return tx.prepareExecContext(ctx, tx.tx, params, name)
}

func (tx *TemplateTxDB) SelectScanFunc(params any, scanFunc any, name ...any) {
	tx.selectScanFunc(context.Background(), tx.tx, params, scanFunc, name)
}
func (tx *TemplateTxDB) SelectScanFuncContext(ctx context.Context, params any, scanFunc any, name ...any) {
	tx.selectScanFunc(ctx, tx.tx, params, scanFunc, name)
}

func (tx *TemplateTxDB) selectByType(ctx context.Context, params any, t reflect.Type, name ...any) reflect.Value {
	return tx.selectCommon(ctx, tx.tx, params, t, 0, name)
}

func (db *TemplateTxDB) RawExec(query string, args ...any) (sql.Result, error) {
	return db.tx.Exec(query, args...)
}

func (db *TemplateTxDB) RawExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.tx.ExecContext(ctx, query, args...)
}

func (db *TemplateTxDB) RawQuery(query string, args ...any) (*sql.Rows, error) {
	return db.tx.Query(query, args...)
}

func (db *TemplateTxDB) RawQueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.tx.QueryContext(ctx, query, args...)
}
