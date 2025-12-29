package tgsql

import (
	"context"
	"database/sql"
	"errors"
)

type recoverPanic struct{}

func (tdb *TgenSql) NewRecover(ctx context.Context) context.Context {
	if _, ok := tdb.FromRecover(ctx); ok {
		return ctx
	}
	isRecoverPanic := false
	return context.WithValue(ctx, recoverPanic{}, &isRecoverPanic)
}

func (tdb *TgenSql) Recover(ctx context.Context, err *error) {
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

type enableSqlTxKey struct{}
type sqlTxKey struct{}

func NewSqlTx(ctx context.Context, tx *sql.Tx) context.Context {
	ctx = context.WithValue(ctx, enableSqlTxKey{}, true)
	return context.WithValue(ctx, sqlTxKey{}, tx)
}

func GetEnableSqlTx(ctx context.Context) bool {
	if enable, ok := ctx.Value(enableSqlTxKey{}).(bool); ok && enable {
		return true
	}
	return false
}

func FromSqlTx(ctx context.Context) (tx *sql.Tx, ok bool) {
	tx, ok = ctx.Value(sqlTxKey{}).(*sql.Tx)
	return
}

func (tdb *TgenSql) Begin(ctx context.Context) (context.Context, error) {
	return tdb.BeginTx(ctx, nil)
}

func (tdb *TgenSql) BeginTx(ctx context.Context, opts *sql.TxOptions) (context.Context, error) {
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
func (tdb *TgenSql) AutoCommit(ctx context.Context, err *error) {
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

func (tdb *TgenSql) Rollback(ctx context.Context) error {
	tx, ok := FromSqlTx(ctx)
	if ok {
		return tx.Rollback()
	}
	return nil
}

func (tdb *TgenSql) Commit(ctx context.Context) error {
	tx, ok := FromSqlTx(ctx)
	if ok {
		return tx.Commit()
	}
	return nil
}
