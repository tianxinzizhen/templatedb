package templatedb

import (
	"context"
	"database/sql"
	"embed"
	"strings"

	"github.com/tianxinzizhen/templatedb/load/xml"
	"github.com/tianxinzizhen/templatedb/template"
)

type DefaultDB struct {
	sqlDB                   *sql.DB
	template                map[string]*template.Template
	delimsLeft, delimsRight string
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

func (db *DefaultDB) parse(parse string, addParseTrees ...func(*template.Template) error) (*template.Template, error) {
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

func (db *DefaultDB) Exec(statement string, params any) (lastInsertId, rowsAffected int, err error) {
	sql, args, err := db.templateBuild(statement, params)
	if err != nil {
		return
	}
	result, err := db.sqlDB.Exec(sql, args...)
	if err != nil {
		return
	}
	lastid, err := result.LastInsertId()
	if err != nil {
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return
	}
	return int(lastid), int(affected), nil
}

func (db *DefaultDB) ExecMulti(statement string, param any) (rowsAffected int, err error) {
	sqls := strings.Split(statement, ";")
	for _, sql := range sqls {
		if len(strings.Trim(sql, "\t\n\f\r ")) == 0 {
			continue
		}
		execSql, args, err := db.templateBuild(sql, param)
		if err != nil {
			return 0, err
		}
		result, err := db.sqlDB.Exec(execSql, args)
		if err != nil {
			return 0, err
		}
		itemAffected, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}
		rowsAffected += int(itemAffected)
	}
	return 0, nil
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

func (tx *TemplateTxDB) Rollback() (err error) {
	return tx.tx.Rollback()
}
func (tx *TemplateTxDB) Commit() (err error) {
	return tx.tx.Commit()
}
func (tx *TemplateTxDB) AutoCommit(err *error) {
	if *err != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}
}

func (tx *TemplateTxDB) Exec(statement string, params any) (lastInsertId, rowsAffected int, err error) {
	sql, args, err := tx.db.templateBuild(statement, params)
	if err != nil {
		return
	}
	result, err := tx.tx.Exec(sql, args...)
	if err != nil {
		return
	}
	lastid, err := result.LastInsertId()
	if err != nil {
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return
	}
	return int(lastid), int(affected), nil
}

func (tx *TemplateTxDB) ExecMulti(statement string, param any) (rowsAffected int, err error) {
	sqls := strings.Split(statement, ";")
	for _, sql := range sqls {
		if len(strings.Trim(sql, "\t\n\f\r ")) == 0 {
			continue
		}
		execSql, args, err := tx.db.templateBuild(sql, param)
		if err != nil {
			return 0, err
		}
		result, err := tx.tx.Exec(execSql, args...)
		if err != nil {
			return 0, err
		}
		itemAffected, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}
		rowsAffected += int(itemAffected)
	}
	return 0, nil
}

func (tx *TemplateTxDB) PrepareBatch(statement string, params ...any) (rowsAffected int, err error) {
	var stmtMaps map[string]*sql.Stmt = make(map[string]*sql.Stmt)
	var tempSql string
	for _, param := range params {
		execSql, args, err := tx.db.templateBuild(statement, param)
		if err != nil {
			return 0, err
		}
		var stmt *sql.Stmt
		if tempSql != execSql {
			tempSql = execSql
			if s, ok := stmtMaps[execSql]; !ok {
				stmt, err = tx.tx.Prepare(execSql)
				if err != nil {
					return 0, err
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
			return 0, err
		}
		batchAffected, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}
		rowsAffected += int(batchAffected)
	}
	return
}
