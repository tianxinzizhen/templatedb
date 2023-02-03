package templatedb

import (
	"context"
	"database/sql"
	"reflect"
)

type UsualTemplateDB struct {
	*DefaultDB
}

func NewUsualTemplateDB(db *DefaultDB) *UsualTemplateDB {
	return &UsualTemplateDB{DefaultDB: db}
}

func (db *UsualTemplateDB) RawExec(query string, args ...any) (sql.Result, error) {
	return db.sqlDB.Exec(query, args...)
}

func (db *UsualTemplateDB) RawExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.sqlDB.ExecContext(ctx, query, args...)
}

func (db *UsualTemplateDB) RawQuery(query string, args ...any) (*sql.Rows, error) {
	return db.sqlDB.Query(query, args...)
}

func (db *UsualTemplateDB) RawQueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.sqlDB.QueryContext(ctx, query, args...)
}

func (db *UsualTemplateDB) Exec(name string, params any) (lastInsertId, rowsAffected int64) {
	return db.exec(context.Background(), db.sqlDB, params, []any{name})
}

func (db *UsualTemplateDB) ExecContext(ctx context.Context, name string, params any) (lastInsertId, rowsAffected int64) {
	return db.exec(ctx, db.sqlDB, params, []any{name})
}

func (db *UsualTemplateDB) PrepareExec(name string, params []any) (rowsAffected int64) {
	return db.prepareExecContext(context.Background(), db.sqlDB, params, []any{name})
}

func (db *UsualTemplateDB) PrepareExecContext(ctx context.Context, name string, params []any) (rowsAffected int64) {
	return db.prepareExecContext(ctx, db.sqlDB, params, []any{name})
}

func (db *UsualTemplateDB) SelectScanFunc(name string, params any, scanFunc any) {
	db.selectScanFunc(context.Background(), db.sqlDB, params, scanFunc, []any{name})
}

func (db *UsualTemplateDB) SelectScanFuncContext(ctx context.Context, name string, params any, scanFunc any) {
	db.selectScanFunc(ctx, db.sqlDB, params, scanFunc, []any{name})
}

func (db *UsualTemplateDB) SelectByModel(name string, params any, model any) any {
	return db.selectCommon(context.Background(), db.sqlDB, params, reflect.TypeOf(model), 0, []any{name}).Interface()
}
func (db *UsualTemplateDB) SelectByModelContext(ctx context.Context, name string, params any, model any) any {
	return db.selectCommon(ctx, db.sqlDB, params, reflect.TypeOf(model), 0, []any{name}).Interface()
}

func (db *UsualTemplateDB) Begin() (*UsualTemplateTxDB, error) {
	return db.BeginTx(context.Background(), nil)
}

func (db *UsualTemplateDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*UsualTemplateTxDB, error) {
	tx, err := db.DefaultDB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &UsualTemplateTxDB{TemplateTxDB: tx}, nil
}

type UsualTemplateTxDB struct {
	*TemplateTxDB
}

func (tx *UsualTemplateTxDB) AutoCommit(errp *error) {
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

func (tx *UsualTemplateTxDB) Exec(name string, params any) (lastInsertId, rowsAffected int64) {
	return tx.exec(context.Background(), tx.tx, params, []any{name})
}

func (tx *UsualTemplateTxDB) ExecContext(ctx context.Context, name string, params any) (lastInsertId, rowsAffected int64) {
	return tx.exec(ctx, tx.tx, params, []any{name})
}

func (tx *UsualTemplateTxDB) PrepareExec(name string, params []any) (rowsAffected int64) {
	return tx.prepareExecContext(context.Background(), tx.tx, params, []any{name})
}

func (tx *UsualTemplateTxDB) PrepareExecContext(ctx context.Context, name string, params []any) (rowsAffected int64) {
	return tx.prepareExecContext(ctx, tx.tx, params, []any{name})
}

func (tx *UsualTemplateTxDB) SelectScanFunc(name string, params any, scanFunc any) {
	tx.selectScanFunc(context.Background(), tx.tx, params, scanFunc, []any{name})
}
func (tx *UsualTemplateTxDB) SelectScanFuncContext(ctx context.Context, name string, params any, scanFunc any) {
	tx.selectScanFunc(ctx, tx.tx, params, scanFunc, []any{name})
}

func (tx *UsualTemplateTxDB) SelectByModel(name string, params any, model any) any {
	return tx.selectCommon(context.Background(), tx.tx, params, reflect.TypeOf(model), 0, []any{name}).Interface()
}

func (tx *UsualTemplateTxDB) SelectByModelContext(ctx context.Context, name string, params any, model any) any {
	return tx.selectCommon(ctx, tx.tx, params, reflect.TypeOf(model), 0, []any{name}).Interface()
}

func (db *UsualTemplateTxDB) RawExec(query string, args ...any) (sql.Result, error) {
	return db.tx.Exec(query, args...)
}

func (db *UsualTemplateTxDB) RawExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.tx.ExecContext(ctx, query, args...)
}

func (db *UsualTemplateTxDB) RawQuery(query string, args ...any) (*sql.Rows, error) {
	return db.tx.Query(query, args...)
}

func (db *UsualTemplateTxDB) RawQueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.tx.QueryContext(ctx, query, args...)
}
