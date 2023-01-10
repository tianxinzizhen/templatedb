package templatedb

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"reflect"
	"runtime"

	"github.com/tianxinzizhen/templatedb/load"
	"github.com/tianxinzizhen/templatedb/load/xml"
	"github.com/tianxinzizhen/templatedb/template"
)

type DefaultDB struct {
	sqlDB                   *sql.DB
	template                map[string]*template.Template
	delimsLeft, delimsRight string
}

func getSkipFuncName(skip int, name []any) string {
	if len(name) > 0 && reflect.TypeOf(name[0]).Kind() == reflect.Func {
		return fmt.Sprintf("%s:%s", runtime.FuncForPC(reflect.ValueOf(name[0]).Pointer()).Name(), fmt.Sprint(name[1:]...))
	}
	pc, _, _, _ := runtime.Caller(skip)
	funcName := runtime.FuncForPC(pc).Name()
	return fmt.Sprintf("%s:%s", funcName, fmt.Sprint(name...))
}

func Delims(delimsLeft, delimsRight string) func(*DefaultDB) {
	return func(db *DefaultDB) {
		db.delimsLeft = delimsLeft
		db.delimsRight = delimsRight
	}
}

func LoadSqlOfXml(sqlDir embed.FS) func(*DefaultDB) error {
	return func(db *DefaultDB) error {
		if db.template == nil {
			db.template = make(map[string]*template.Template)
		}
		err := xml.LoadTemplateStatements(sqlDir, db.template, db.parse)
		if err != nil {
			return err
		}
		return nil
	}
}

func NewDefaultDB(SqlDB *sql.DB, options ...func(*DefaultDB) error) (*DefaultDB, error) {
	db := &DefaultDB{
		sqlDB:      SqlDB,
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

func (db *DefaultDB) Recover(err *error) {
	e := recover()
	if e != nil {
		*err = e.(error)
	}
}

func (db *DefaultDB) parse(parse string, addParseTrees ...load.AddParseTree) (*template.Template, error) {
	templateSql, err := template.New("").Delims(db.delimsLeft, db.delimsRight).Funcs(sqlfunc).Parse(parse)
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
		templateSql, err = db.parse(query)
		if err != nil {
			return
		}
		db.template[query] = templateSql
	}
	return templateSql.ExecuteBuilder(params)
}

func (db *DefaultDB) Exec(params any, name ...any) (lastInsertId, rowsAffected int) {
	statement := getSkipFuncName(2, name)
	sql, args, err := db.templateBuild(statement, params)
	if err != nil {
		panic(err)
	}
	result, err := db.sqlDB.Exec(sql, args...)
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
	return int(lastid), int(affected)
}

func (db *DefaultDB) Begin() (*TemplateTxDB, error) {
	tx, err := db.sqlDB.Begin()
	if err != nil {
		return nil, err
	}
	return &TemplateTxDB{db: db, tx: tx}, nil
}

func (db *DefaultDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*TemplateTxDB, error) {
	tx, err := db.sqlDB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &TemplateTxDB{db: db, tx: tx}, nil
}

type TemplateTxDB struct {
	db *DefaultDB
	tx *sql.Tx
}

func (tx *TemplateTxDB) AutoCommit(err *error) {
	if *err != nil {
		tx.tx.Rollback()
	} else {
		e := recover()
		if e != nil {
			*err = e.(error)
			tx.tx.Rollback()
		} else {
			tx.tx.Commit()
		}
	}
}

func (tx *TemplateTxDB) Exec(params any, name ...any) (lastInsertId, rowsAffected int) {
	statement := getSkipFuncName(2, name)
	sql, args, err := tx.db.templateBuild(statement, params)
	if err != nil {
		panic(err)
	}
	result, err := tx.tx.Exec(sql, args...)
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
	return int(lastid), int(affected)
}

func (tx *TemplateTxDB) PrepareExec(params []any, name ...any) (rowsAffected int) {
	statement := getSkipFuncName(2, name)
	var stmtMaps map[string]*sql.Stmt = make(map[string]*sql.Stmt)
	var tempSql string
	for _, param := range params {
		execSql, args, err := tx.db.templateBuild(statement, param)
		if err != nil {
			panic(err)
		}
		var stmt *sql.Stmt
		if tempSql != execSql {
			tempSql = execSql
			if s, ok := stmtMaps[execSql]; !ok {
				stmt, err = tx.tx.Prepare(execSql)
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
		result, err := stmt.Exec(args...)
		if err != nil {
			panic(err)
		}
		batchAffected, err := result.RowsAffected()
		if err != nil {
			panic(err)
		}
		rowsAffected += int(batchAffected)
	}
	return
}
