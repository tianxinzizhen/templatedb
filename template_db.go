package templatedb

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/tianxinzizhen/templatedb/template"
)

type SqlDebug func(ctx context.Context, sql string)

type ValueMethod map[reflect.Type]reflect.Value

var scanValue ValueMethod = make(ValueMethod)

var setValue ValueMethod = make(ValueMethod)

func AddGlobalScanValue(method any) error {
	mt := reflect.TypeOf(method)
	if mt.Kind() != reflect.Func {
		return fmt.Errorf("the incoming value is not a function")
	}
	rt := mt.In(0)
	if rt.Kind() != reflect.Pointer {
		return fmt.Errorf("the incoming value is not a pointer")
	}
	if !(mt.NumIn() == 2 &&
		(mt.NumOut() == 0 ||
			(mt.NumOut() == 1 && mt.Out(0).Implements(errorType)))) {
		return fmt.Errorf("the parameter of this function is set incorrectly")
	}
	if scanValue == nil {
		scanValue = make(ValueMethod)
	}
	if _, ok := scanValue[rt]; !ok {
		scanValue[rt] = reflect.ValueOf(method)
	} else {
		return fmt.Errorf("the type of function is set")
	}
	return nil
}

func AddGlobalSetValue(method any) error {
	mt := reflect.TypeOf(method)
	if mt.Kind() != reflect.Func {
		return fmt.Errorf("the incoming value is not a function")
	}
	rt := mt.In(0)
	if rt.Kind() != reflect.Pointer {
		return fmt.Errorf("the incoming value is not a pointer")
	}
	if !(mt.NumIn() == 1 &&
		((mt.NumOut() == 1 && !mt.Out(0).Implements(errorType)) ||
			(mt.NumOut() == 2 && mt.Out(1).Implements(errorType)))) {
		return fmt.Errorf("the parameter of this function is set incorrectly")
	}
	if setValue == nil {
		setValue = make(ValueMethod)
	}
	if _, ok := setValue[rt]; !ok {
		setValue[rt] = reflect.ValueOf(method)
	} else {
		return fmt.Errorf("the type of function is set")
	}
	return nil
}

type TemplateDb struct {
	db        *sql.DB
	sqlDebug  SqlDebug
	template  map[uintptr]*template.Template //sql t offset struct
	scanValue ValueMethod
	setValue  ValueMethod
}

func NewTemplateDb(ops ...Option) *TemplateDb {
	td := &TemplateDb{}
	// default set global value
	WithScanValue(scanValue)
	WithSetValue(setValue)
	for _, op := range ops {
		op(td)
	}
	return td
}

type Option func(*TemplateDb)

func WithDB(db *sql.DB) Option {
	return func(td *TemplateDb) {
		td.db = db
	}
}

func WithSqlDebug(sqlDebug SqlDebug) Option {
	return func(td *TemplateDb) {
		td.sqlDebug = sqlDebug
	}
}

func WithScanValue(scanValue ValueMethod) Option {
	return func(td *TemplateDb) {
		if td.scanValue == nil {
			td.scanValue = make(ValueMethod)
		}
		if scanValue != nil {
			for k, v := range scanValue {
				td.scanValue[k] = v
			}
		}
	}
}

func WithSetValue(setValue ValueMethod) Option {
	return func(td *TemplateDb) {
		if td.setValue == nil {
			td.setValue = make(ValueMethod)
		}
		if setValue != nil {
			for k, v := range setValue {
				td.setValue[k] = v
			}
		}
	}
}
func (td *TemplateDb) AddScanValue(method any) error {
	mt := reflect.TypeOf(method)
	if mt.Kind() != reflect.Func {
		return fmt.Errorf("the incoming value is not a function")
	}
	rt := mt.In(0)
	if rt.Kind() != reflect.Pointer {
		return fmt.Errorf("the incoming value is not a pointer")
	}
	if !(mt.NumIn() == 2 &&
		(mt.NumOut() == 0 ||
			(mt.NumOut() == 1 && mt.Out(0).Implements(errorType)))) {
		return fmt.Errorf("the parameter of this function is set incorrectly")
	}
	td.scanValue[rt] = reflect.ValueOf(method)
	return nil
}

func (td *TemplateDb) AddSetValue(method any) error {
	mt := reflect.TypeOf(method)
	if mt.Kind() != reflect.Func {
		return fmt.Errorf("the incoming value is not a function")
	}
	rt := mt.In(0)
	if rt.Kind() != reflect.Pointer {
		return fmt.Errorf("the incoming value is not a pointer")
	}
	if !(mt.NumIn() == 1 &&
		((mt.NumOut() == 1 && !mt.Out(0).Implements(errorType)) ||
			(mt.NumOut() == 2 && mt.Out(1).Implements(errorType)))) {
		return fmt.Errorf("the parameter of this function is set incorrectly")
	}
	td.setValue[rt] = reflect.ValueOf(method)
	return nil
}
