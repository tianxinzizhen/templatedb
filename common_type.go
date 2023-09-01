package templatedb

import (
	"context"
	"reflect"
)

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
	ResultType  = reflect.TypeOf((*Result)(nil))
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

type Result struct {
	LastInsertId int64
	RowsAffected int64
}

type ExecResult struct {
	LastInsertId int64
	RowsAffected int64
	err          error
}
