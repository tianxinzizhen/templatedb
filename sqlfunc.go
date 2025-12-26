package templatedb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/tianxinzizhen/templatedb/sqlwrite"
	"github.com/tianxinzizhen/templatedb/template"

	"github.com/tianxinzizhen/templatedb/util"
)

var sqlFunc template.FuncMap = make(template.FuncMap)

var SqlEscapeBytesBackslash = false

func comma(iVal reflect.Value) (*sqlwrite.SqlWrite, error) {
	i, isNil := util.Indirect(iVal)
	if isNil {
		return nil, fmt.Errorf("comma sql function in paramter is nil")
	}
	sqw := &sqlwrite.SqlWrite{}
	var commaPrint bool
	switch i.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		commaPrint = i.Int() > 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		commaPrint = i.Uint() > 0
	default:
		return nil, nil
	}
	if commaPrint {
		sqw.WriteString(",")
	} else {
		sqw.WriteString("")
	}
	return sqw, nil
}

func params(list ...reflect.Value) *sqlwrite.SqlWrite {
	sqw := &sqlwrite.SqlWrite{}
	for i, v := range list {
		if i > 0 {
			sqw.WriteString(",")
		}
		sqw.AddParam("? ", v.Interface())
	}
	return sqw
}

func like(param reflect.Value) *sqlwrite.SqlWrite {
	sqw := &sqlwrite.SqlWrite{}
	p := fmt.Sprint(param)
	lb := strings.Builder{}
	if !strings.HasPrefix(p, "%") {
		lb.WriteByte('%')
	}
	lb.WriteString(p)
	if !strings.HasSuffix(p, "%") {
		lb.WriteByte('%')
	}
	sqw.AddParam("like ?", lb.String())
	return sqw
}

func likeRight(param reflect.Value) *sqlwrite.SqlWrite {
	sqw := &sqlwrite.SqlWrite{}
	p := fmt.Sprint(param)
	lb := strings.Builder{}
	lb.WriteString(p)
	if !strings.HasSuffix(p, "%") {
		lb.WriteByte('%')
	}
	sqw.AddParam("like ?", lb.String())
	return sqw
}
func likeLeft(param reflect.Value) *sqlwrite.SqlWrite {
	sqw := &sqlwrite.SqlWrite{}
	p := fmt.Sprint(param)
	lb := strings.Builder{}
	if !strings.HasPrefix(p, "%") {
		lb.WriteByte('%')
	}
	lb.WriteString(p)
	sqw.AddParam("like ?", lb.String())
	return sqw
}

func marshal(list ...reflect.Value) (*sqlwrite.SqlWrite, error) {
	sqw := &sqlwrite.SqlWrite{}
	for i, v := range list {
		if i > 0 {
			sqw.WriteString(",")
		}
		vi := v.Interface()
		mJson, err := json.Marshal(vi)
		if err != nil {
			return nil, err
		}
		sqw.AddParam("? ", string(mJson))
	}
	return sqw, nil
}

func SqlEscape(arg any) (sql string, err error) {
	return util.GetNoneEscapeSql(arg, SqlEscapeBytesBackslash)
}

func SqlInterpolateParams(query string, arg []any) (sql string, err error) {
	return util.InterpolateParams(query, arg, SqlEscapeBytesBackslash)
}

func jsonTagAsFieldName(tag reflect.StructTag, fieldName string) bool {
	if asName, ok := tag.Lookup("json"); ok {
		if asName == "-" {
			return false
		}
		fName, _, _ := strings.Cut(asName, ",")
		if fieldName == fName {
			return true
		}
	}
	if asName, ok := tag.Lookup("as"); ok {
		if fieldName == asName {
			return true
		}
	}
	return false
}

func getFieldByTag(t reflect.Type, fieldName string, scanNum map[string]int) (f reflect.StructField, ok bool) {
	for i := 0; i < t.NumField(); i++ {
		tf := t.Field(i)
		if jsonTagAsFieldName(tf.Tag, fieldName) {
			return tf, true
		}
		if tf.Anonymous && tf.Type.Kind() == reflect.Struct {
			f, ok = getFieldByTag(tf.Type, fieldName, scanNum)
			if ok {
				if scanNum != nil {
					if _, ok := scanNum[f.Name]; ok {
						if i <= scanNum[f.Name] {
							continue
						} else {
							scanNum[f.Name] = i
						}
					} else {
						scanNum[f.Name] = i
					}
				}
				f.Index = append(tf.Index, f.Index...)
				return
			}
		}
	}
	return
}
func DefaultGetFieldByName(t reflect.Type, fieldName string, scanNum map[string]int) (f reflect.StructField, ok bool) {
	tField, ok := t.FieldByName(fieldName)
	if ok {
		return tField, ok
	}
	f, ok = getFieldByTag(t, fieldName, scanNum)
	return
}

func init() {
	//sql 函数的加载
	AddTemplateFunc("comma", comma)
	AddTemplateFunc("like", like)
	AddTemplateFunc("liker", likeRight)
	AddTemplateFunc("likel", likeLeft)
	AddTemplateFunc("param", params)
	AddTemplateFunc("marshal", marshal)
	AddTemplateFunc("json", marshal)
}

func AddTemplateFunc(key string, funcMethod any) error {
	if _, ok := sqlFunc[key]; ok {
		return fmt.Errorf("add template func[%s] already exists ", key)
	} else {
		sqlFunc[key] = funcMethod
	}
	return nil
}

var MaxStackLen = 50

type sqlDB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}
