package templatedb

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/tianxinzizhen/templatedb/scanner"
	"github.com/tianxinzizhen/templatedb/template"

	"github.com/tianxinzizhen/templatedb/util"
)

var sqlFunc template.FuncMap = make(template.FuncMap)

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
func inParam(list reflect.Value, fieldNames ...any) (string, []any, error) {
	list, isNil := util.Indirect(list)
	if isNil {
		return "", nil, fmt.Errorf("inParam sql function in paramter list is nil")
	}
	fieldName := fmt.Sprint(fieldNames...)
	if list.Kind() == reflect.Slice || list.Kind() == reflect.Array {
		sb := strings.Builder{}
		sb.WriteString("in (")
		var args []any = make([]any, list.Len())
		for i := 0; i < list.Len(); i++ {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteByte('?')
			item, isNil := util.Indirect(list.Index(i))
			if isNil {
				continue
			}
			if !item.IsValid() {
				continue
			}
			switch item.Kind() {
			case reflect.Struct:
				tField, ok := template.GetFieldByName(item.Type(), fieldName)
				if ok {
					field, err := item.FieldByIndexErr(tField.Index)
					if err != nil {
						return "", nil, err
					}
					args[i] = field.Interface()
				} else {
					return "", nil, fmt.Errorf("in params : The attribute %s was not found in the structure %s.%s", fieldName, item.Type().PkgPath(), item.Type().Name())
				}
			case reflect.Map:
				if item.Type().Key().Kind() == reflect.String {
					fieldValue := item.MapIndex(reflect.ValueOf(fieldName))
					if fieldValue.IsValid() {
						args[i] = fieldValue.Interface()
					} else {
						return "", nil, fmt.Errorf("in params : fieldValue in map[%s] IsValid", fieldName)
					}
				} else {
					return "", nil, fmt.Errorf("in params : Map key Type is not string")
				}
			default:
				args[i] = item.Interface()
			}
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
	return false
}

func JsonConvertStruct(field reflect.Value, src any) error {
	if src == nil {
		return nil
	}
	if field.Kind() == reflect.Struct {
		if field.Kind() == reflect.Pointer {
			field.Set(reflect.New(field.Type().Elem()))
		} else {
			field = field.Addr()
		}
		return json.Unmarshal(src.([]byte), field.Interface())
	} else {
		return scanner.ConvertAssign(field.Addr().Interface(), src)
	}

}

func init() {
	//sql 函数的加载
	AddTemplateFunc("comma", comma)
	AddTemplateFunc("in", inParam)
	AddTemplateFunc("like", like)
	AddTemplateFunc("param", params)
	AddTemplateFunc("sqlEscape", sqlEscape)
	AddTemplateFunc("orNull", orNull)
	AddTemplateFunc("marshal", marshal)
	//模版@#号字符串拼接时对字段值转化成sql字符串函数
	template.SqlEscape = SqlEscape
	//使用tag为字段取别名
	template.TagAsFieldName = JsonTagAsFieldName
	//mysql的json字段处理
	AddScanConvertDatabaseTypeFunc("JSON", JsonConvertStruct)
}

func AddTemplateFunc(key string, funcMethod any) {
	sqlFunc[key] = funcMethod
}
