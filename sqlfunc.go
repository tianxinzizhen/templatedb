package templatedb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

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
	args[0] = lb.String()
	return " like ? ", args
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
			args[i] = string(mJson)
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
	//模版@#号字符串拼接时对字段值转化成sql字符串函数
	template.SqlEscape = SqlEscape
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

type commonSqlFunc struct {
	getFieldByName func(t reflect.Type, fieldName string, scanNum map[string]int) (f reflect.StructField, ok bool)
}

func GetCommonSqlFunc(getFieldByName func(t reflect.Type, fieldName string, scanNum map[string]int) (f reflect.StructField, ok bool)) template.FuncMap {
	cf := &commonSqlFunc{getFieldByName: getFieldByName}
	if cf.getFieldByName == nil {
		cf.getFieldByName = DefaultGetFieldByName
	}
	sqlCommonFunc := make(template.FuncMap)
	sqlCommonFunc["in"] = cf.inParam
	sqlCommonFunc["value"] = cf.value
	sqlCommonFunc["values"] = cf.values
	sqlCommonFunc["set"] = cf.set
	sqlCommonFunc["setl"] = cf.setl
	sqlCommonFunc["setr"] = cf.setr
	sqlCommonFunc["if_and"] = cf.if_and
	return sqlCommonFunc
}
func indirect(v reflect.Value) (rv reflect.Value, isNil bool) {
	for ; v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface; v = v.Elem() {
		if v.IsNil() {
			return v, true
		}
	}
	return v, false
}

func (s *commonSqlFunc) inParam(list reflect.Value, fieldNames ...string) (string, []any, error) {
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
					tField, ok := s.getFieldByName(sv.Type(), fieldName, nil)
					if ok {
						field, err := sv.FieldByIndexErr(tField.Index)
						if err != nil {
							return "", nil, err
						}
						if i == len(fieldNames)-1 {
							if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
								for i := 0; i < field.Len(); i++ {
									val := field.Index(i).Interface()
									if _, ok := exists[val]; !ok {
										exists[val] = struct{}{}
										args = append(args, val)
									}
								}
							} else {
								val := field.Interface()
								if _, ok := exists[val]; !ok {
									exists[val] = struct{}{}
									args = append(args, val)
								}
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

func (s *commonSqlFunc) value(val reflect.Value, value string) (string, []any, error) {
	values := strings.Split(value, ",")
	sqlBuilder := strings.Builder{}
	args := make([]any, 0, len(values))
	val, isNil := indirect(val)
	if isNil {
		return "", nil, fmt.Errorf("value sql function in(0) is nil")
	}
	for i, column := range values {
		if i > 0 {
			sqlBuilder.WriteRune(',')
		}
		ps := "?"
		column = strings.TrimSpace(column)
		switch column {
		case "CURRENT_TIMESTAMP":
			sqlBuilder.WriteString(column)
		default:
			switch val.Kind() {
			case reflect.Struct:
				tField, ok := s.getFieldByName(val.Type(), column, nil)
				if ok {
					field, err := val.FieldByIndexErr(tField.Index)
					if err != nil {
						return "", nil, err
					}
					args = append(args, field.Interface())
				}
			case reflect.Map:
				if val.Type().Key().Kind() == reflect.String {
					args = append(args, val.MapIndex(reflect.ValueOf(column)).Interface())
				}
			}
			sqlBuilder.WriteString(ps)
		}
	}
	return sqlBuilder.String(), args, nil
}

func (s *commonSqlFunc) values(sVal reflect.Value, value string) (string, []any, error) {
	values := strings.Split(value, ",")
	sqlBuilder := strings.Builder{}
	sqlBuilder.WriteString(" VALUES ")
	if sVal.Kind() != reflect.Slice {
		return "", nil, fmt.Errorf("values sql function in(0) is not slice")
	}
	args := make([]any, 0, len(values)*sVal.Len())
	for i := 0; i < sVal.Len(); i++ {
		if i > 0 {
			sqlBuilder.WriteRune(',')
		}
		sqlBuilder.WriteString("(")
		val, isNil := indirect(sVal.Index(i))
		if isNil {
			return "", nil, fmt.Errorf("value sql function in(0) is nil")
		}
		for j, column := range values {
			if j > 0 {
				sqlBuilder.WriteRune(',')
			}
			ps := "?"
			column = strings.TrimSpace(column)
			switch column {
			case "CURRENT_TIMESTAMP":
				sqlBuilder.WriteString(column)
			default:
				switch val.Kind() {
				case reflect.Struct:
					tField, ok := s.getFieldByName(val.Type(), column, nil)
					if ok {
						field, err := val.FieldByIndexErr(tField.Index)
						if err != nil {
							return "", nil, err
						}
						args = append(args, field.Interface())
					} else {
						return "", nil, fmt.Errorf("column:%s in struct %v not found", column, val.Type().Name())
					}
				case reflect.Map:
					if val.Type().Key().Kind() == reflect.String {
						args = append(args, val.MapIndex(reflect.ValueOf(column)).Interface())
					} else {
						return "", nil, fmt.Errorf("column:%s in map key is not string", column)
					}
				}
				sqlBuilder.WriteString(ps)
			}
		}
		sqlBuilder.WriteString(")")
	}
	return sqlBuilder.String(), args, nil
}

func (s *commonSqlFunc) set(val reflect.Value, action []string, value string) (string, []any, error) {
	val, isNil := indirect(val)
	if isNil {
		return "", nil, fmt.Errorf("value sql function in(0) is nil")
	}
	actionMap := make(map[string]struct{})
	for _, v := range action {
		actionMap[v] = struct{}{}
	}
	values := strings.Split(value, ",")
	sqlBuilder := strings.Builder{}
	var args []any
	for _, v := range values {
		v = strings.TrimSpace(v)
		if _, ok := actionMap[v]; !ok {
			continue
		}
		ps := "?"
		var arg any
		switch val.Kind() {
		case reflect.Struct:
			tField, ok := s.getFieldByName(val.Type(), v, nil)
			if ok {
				field, err := val.FieldByIndexErr(tField.Index)
				if err != nil {
					return "", nil, err
				}
				arg = field.Interface()
			}
		case reflect.Map:
			if val.Type().Key().Kind() == reflect.String {
				arg = val.MapIndex(reflect.ValueOf(v)).Interface()
			}
		}
		if len(args) > 0 {
			sqlBuilder.WriteRune(',')
		}
		sqlBuilder.WriteString(v)
		sqlBuilder.WriteString("=")
		sqlBuilder.WriteString(ps)
		args = append(args, arg)
	}
	return sqlBuilder.String(), args, nil
}

func (s *commonSqlFunc) if_and(val reflect.Value, value string) (string, []any, error) {
	val, isNil := indirect(val)
	if isNil {
		return "", nil, fmt.Errorf("value sql function in(0) is nil")
	}
	values := strings.Split(value, ",")
	sqlBuilder := strings.Builder{}
	var args []any
	for _, column := range values {
		column = strings.TrimSpace(column)
		var preFuzzy, sufFuzzy bool
		if strings.HasPrefix(column, "%") {
			preFuzzy = true
			column = strings.TrimPrefix(column, "%")
		}
		if strings.HasSuffix(column, "%") {
			sufFuzzy = true
			column = strings.TrimSuffix(column, "%")
		}
		var fieldByName string = column
		if _, after, found := strings.Cut(column, "."); found {
			fieldByName = after
		}
		ps := "?"
		var arg any
		switch val.Kind() {
		case reflect.Struct:
			tField, ok := s.getFieldByName(val.Type(), fieldByName, nil)
			if ok {
				field, err := val.FieldByIndexErr(tField.Index)
				if err != nil {
					return "", nil, err
				}
				arg = field.Interface()
				if truth, _ := template.IsTrue(arg); !truth {
					continue
				}
			} else {
				continue
			}
		case reflect.Map:
			if val.Type().Key().Kind() == reflect.String {
				arg = val.MapIndex(reflect.ValueOf(fieldByName)).Interface()
				if truth, _ := template.IsTrue(arg); !truth {
					continue
				}
			} else {
				continue
			}
		}
		sqlBuilder.WriteString(" and ")
		sqlBuilder.WriteString(column)
		if preFuzzy || sufFuzzy {
			sqlBuilder.WriteString(" like ")
			sqlBuilder.WriteString(ps)
			argBuilder := &strings.Builder{}
			if preFuzzy {
				argBuilder.WriteString("%")
			}
			argBuilder.WriteString(fmt.Sprint(arg))
			if sufFuzzy {
				argBuilder.WriteString("%")
			}
			arg = argBuilder.String()
		} else {
			sqlBuilder.WriteString("=")
			sqlBuilder.WriteString(ps)
		}
		args = append(args, arg)
	}
	return sqlBuilder.String(), args, nil
}

func (s *commonSqlFunc) setl(val reflect.Value, action []string, value string) (string, []any, error) {
	sql, args, err := s.set(val, action, value)
	if len(args) > 0 {
		sql = fmt.Sprintf(",%s", sql)
	}
	return sql, args, err
}
func (s *commonSqlFunc) setr(val reflect.Value, action []string, value string) (string, []any, error) {
	sql, args, err := s.set(val, action, value)
	if len(args) > 0 {
		sql = fmt.Sprintf("%s,", sql)
	}
	return sql, args, err
}
