package tgsql

import "context"

type recordSqlKey struct{}
type RecordSqlItem struct {
	Sql  string
	Args []any
}
type RecordSql struct {
	List []RecordSqlItem
}

func (tdb *TgenSql) FromRecordSql(ctx context.Context) (*RecordSql, bool) {
	if ctx == nil {
		return nil, false
	}
	recordSql, ok := ctx.Value(recordSqlKey{}).(*RecordSql)
	return recordSql, ok
}

func (tdb *TgenSql) NewRecordSql(ctx context.Context) context.Context {
	if _, ok := tdb.FromRecordSql(ctx); ok {
		return ctx
	}
	return context.WithValue(ctx, recordSqlKey{}, &RecordSql{})
}
