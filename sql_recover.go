package tgsql

import (
	"context"
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
