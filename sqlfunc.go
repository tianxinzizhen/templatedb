package templatedb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/tianxinzizhen/templatedb/scanner"
	"github.com/tianxinzizhen/templatedb/template"

	"github.com/tianxinzizhen/templatedb/util"
)

var sqlFunc template.FuncMap = make(template.FuncMap)

var sqlParamType map[reflect.Type]struct{} = make(map[reflect.Type]struct{})

var SqlEscapeBytesBackslash = false

func comma(iVal reflect.Value) (string, error) {
	i, isNil := util.Indirect(iVal)
	if isNil {
		return "", fmt.Errorf("comma sql function in paramter is nil")
	}
	var commaPrint bool
	switch i.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		commaPrint = i.Int() > 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		commaPrint = i.Uint() > 0
	default:
		return "", nil
	}
	if commaPrint {
		return ",", nil
	} else {
		return "", nil
	}
}
func inParam(list reflect.Value, fieldNames ...string) (string, []any, error) {
	list, isNil := util.Indirect(list)
	if isNil {
		return "in(NULL)", nil, nil
	}
	if list.Kind() == reflect.Slice || list.Kind() == reflect.Array {
		sb := strings.Builder{}
		sb.WriteString("in (")
		var args []any = make([]any, 0, list.Len())
		exists := make(map[any]any)
		for i := 0; i < list.Len(); i++ {
			item, isNil := util.Indirect(list.Index(i))
			if isNil {
				continue
			}
			if !item.IsValid() {
				continue
			}
			switch item.Kind() {
			case reflect.Struct:
				sv := item
				for i := 0; i < len(fieldNames); i++ {
					fieldName := fieldNames[i]
					sv, isNil = util.Indirect(sv)
					if isNil {
						args = append(args, nil)
						break
					}
					tField, ok := template.GetFieldByName(sv.Type(), fieldName, nil)
					if ok {
						field, err := sv.FieldByIndexErr(tField.Index)
						if err != nil {
							return "", nil, err
						}
						if i == len(fieldNames)-1 {
							val := field.Interface()
							if _, ok := exists[val]; !ok {
								exists[val] = struct{}{}
								args = append(args, val)
							}
						} else {
							sv = field
						}
					} else {
						return "", nil, fmt.Errorf("in params : The attribute %s was not found in the structure %s.%s", fieldName, item.Type().PkgPath(), item.Type().Name())
					}
				}
			case reflect.Map:
				if item.Type().Key().Kind() == reflect.String {
					fieldValue := item.MapIndex(reflect.ValueOf(fmt.Sprint(fieldNames)))
					if fieldValue.IsValid() {
						val := fieldValue.Interface()
						if _, ok := exists[val]; !ok {
							exists[val] = struct{}{}
							args = append(args, val)
						}
					} else {
						return "", nil, fmt.Errorf("in params : fieldValue in map[%s] IsValid", fmt.Sprint(fieldNames))
					}
				} else {
					return "", nil, fmt.Errorf("in params : Map key Type is not string")
				}
			default:
				val := item.Interface()
				if _, ok := exists[val]; !ok {
					exists[val] = struct{}{}
					args = append(args, val)
				}
			}
		}
		for i := range args {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteByte('?')
		}
		if len(args) == 0 {
			sb.WriteString("NULL")
		}
		sb.WriteString(")")
		return sb.String(), args, nil
	} else {
		return "", nil, errors.New("in params : variables are not arrays or slices")
	}
}
func params(list ...reflect.Value) (string, []any) {
	sb := strings.Builder{}
	var args []any = make([]any, len(list))
	for i, v := range list {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteByte('?')
		args[i] = v.Interface()
	}
	return sb.String(), args
}

func like(param reflect.Value) (string, []any) {
	var args []any = make([]any, 1)
	p := fmt.Sprint(param)
	lb := strings.Builder{}
	if !strings.HasPrefix(p, "%") {
		lb.WriteByte('%')
	}
	lb.WriteString(p)
	if !strings.HasSuffix(p, "%") {
		lb.WriteByte('%')
	}
	args[0] = lb.String()
	return " like ? ", args
}

func likeRight(param reflect.Value) (string, []any) {
	var args []any = make([]any, 1)
	p := fmt.Sprint(param)
	lb := strings.Builder{}
	if !strings.HasPrefix(p, "%") {
		lb.WriteByte('%')
	}
	lb.WriteString(p)
	if !strings.HasSuffix(p, "%") {
		lb.WriteByte('%')
	}
	args[0] = lb.String()
	return " like ? ", args
}
func likeLeft(param reflect.Value) (string, []any) {
	var args []any = make([]any, 1)
	p := fmt.Sprint(param)
	lb := strings.Builder{}
	if !strings.HasPrefix(p, "%") {
		lb.WriteByte('%')
	}
	lb.WriteString(p)
	if !strings.HasSuffix(p, "%") {
		lb.WriteByte('%')
	}
	args[0] = lb.String()
	return " like ? ", args
}

func sqlEscape(list ...reflect.Value) (string, error) {
	sb := strings.Builder{}
	for i, v := range list {
		if i > 0 {
			sb.WriteByte(',')
		}
		param, err := util.GetNoneEscapeSql(v.Interface(), SqlEscapeBytesBackslash)
		if err != nil {
			return "", err
		}
		sb.WriteString(param)
	}
	return sb.String(), nil
}

func orNull(list ...reflect.Value) (string, []any) {
	sb := strings.Builder{}
	var args []any = make([]any, len(list))
	for i, v := range list {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteByte('?')
		vi := v.Interface()
		isTrue, _ := template.IsTrue(vi)
		if isTrue {
			args[i] = vi
		}
	}
	return sb.String(), args
}

func marshal(list ...reflect.Value) (string, []any, error) {
	sb := strings.Builder{}
	var args []any = make([]any, len(list))
	for i, v := range list {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteByte('?')
		vi := v.Interface()
		isTrue, _ := template.IsTrue(vi)
		if isTrue {
			mJson, err := json.Marshal(vi)
			if err != nil {
				return "", nil, err
			}
			args[i] = mJson
		}
	}
	return sb.String(), args, nil
}

func SqlEscape(arg any) (sql string, err error) {
	return util.GetNoneEscapeSql(arg, SqlEscapeBytesBackslash)
}

func SqlInterpolateParams(query string, arg []any) (sql string, err error) {
	return util.InterpolateParams(query, arg, SqlEscapeBytesBackslash)
}
func JsonTagAsFieldName(tag reflect.StructTag, fieldName string) bool {
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

func JsonConvertStruct(field reflect.Value, src any) error {
	if src == nil {
		return nil
	}
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}
	if field.Kind() == reflect.Slice {
		if field.IsNil() {
			field.Set(reflect.MakeSlice(field.Type(), 0, 10))
		}
	}
	if field.Kind() == reflect.Map {
		if field.IsNil() {
			field.Set(reflect.MakeMap(field.Type()))
		}
	}
	if field.Kind() == reflect.Struct || field.Kind() == reflect.Slice || field.Kind() == reflect.Map {
		return json.Unmarshal(src.([]byte), field.Addr().Interface())
	} else {
		return scanner.ConvertAssign(field.Addr().Interface(), src)
	}

}

func init() {
	//sql 函数的加载
	AddTemplateFunc("comma", comma)
	AddTemplateFunc("in", inParam)
	AddTemplateFunc("like", like)
	AddTemplateFunc("liker", likeRight)
	AddTemplateFunc("likel", likeLeft)
	AddTemplateFunc("param", params)
	AddTemplateFunc("sqlEscape", sqlEscape)
	AddTemplateFunc("orNull", orNull)
	AddTemplateFunc("marshal", marshal)
	AddTemplateFunc("json", marshal)
	//模版@#号字符串拼接时对字段值转化成sql字符串函数
	template.SqlEscape = SqlEscape
	//使用tag为字段取别名
	template.TagAsFieldName = JsonTagAsFieldName
	//mysql的json字段处理
	AddScanConvertDatabaseTypeFunc("JSON", JsonConvertStruct)
	//添加时间为sql参数类型
	AddSqlParamType(reflect.TypeOf(time.Time{}))
}

func AddTemplateFunc(key string, funcMethod any) error {
	if _, ok := sqlFunc[key]; ok {
		return fmt.Errorf("add template func[%s] already exists ", key)
	} else {
		sqlFunc[key] = funcMethod
	}
	return nil
}
func AddSqlParamType(t reflect.Type) {
	sqlParamType[t] = struct{}{}
}

var LogPrintf func(format string, a ...any) (n int, err error)

func recoverPrintf(err error) {
	if LogPrintf != nil && err != nil {
		var pc []uintptr = make([]uintptr, MaxStackLen)
		n := runtime.Callers(3, pc[:])
		frames := runtime.CallersFrames(pc[:n])
		LogPrintf("%s \n", err)
		for frame, more := frames.Next(); more; frame, more = frames.Next() {
			LogPrintf("%s:%d \n", frame.File, frame.Line)
		}
	}
}

var MaxStackLen = 50

type sqlDB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}
