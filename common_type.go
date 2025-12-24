package templatedb

import (
	"context"
	"database/sql"
	"reflect"
)

var (
	contextType   = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType     = reflect.TypeOf((*error)(nil)).Elem()
	sqlResultType = reflect.TypeOf((*sql.Result)(nil)).Elem()
)

type Operation int

const (
	ExecAction Operation = iota
	PrepareAction
	SelectAction
	SelectScanAction
	ExecNoResultAction
)

type LoadType int

const (
	LoadXML LoadType = iota
	LoadComment
)
