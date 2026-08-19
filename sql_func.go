package tgsql

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/tianxinzizhen/tgsql/template"

	"github.com/tianxinzizhen/tgsql/util"
)

type SqlFunc struct {
	Args []any
}

func (*SqlFunc) sqlStr(strs ...string) (string, error) {
	return strings.Join(strs, ""), nil
}

func (sq *SqlFunc) comma(iVal reflect.Value) (string, error) {
	i, isNil := util.Indirect(iVal)
	if isNil {
		return "", fmt.Errorf("comma sql function in paramter is nil")
	}
	sb := &strings.Builder{}
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
		sb.WriteString(",")
	} else {
		sb.WriteString("")
	}
	return sb.String(), nil
}

func (sq *SqlFunc) params(list ...reflect.Value) string {
	sb := &strings.Builder{}
	for i, v := range list {
		if i > 0 {
			sb.WriteString(",")
		}
		sq.Args = append(sq.Args, v.Interface())
		sb.WriteString("?")
	}
	return sb.String()
}

func (sq *SqlFunc) like(param reflect.Value) string {
	p := fmt.Sprint(param)
	lb := strings.Builder{}
	if !strings.HasPrefix(p, "%") {
		lb.WriteByte('%')
	}
	lb.WriteString(p)
	if !strings.HasSuffix(p, "%") {
		lb.WriteByte('%')
	}
	sq.Args = append(sq.Args, p)
	return "like ?"
}

func (sq *SqlFunc) likeRight(param reflect.Value) string {
	p := fmt.Sprint(param)
	lb := strings.Builder{}
	lb.WriteString(p)
	if !strings.HasSuffix(p, "%") {
		lb.WriteByte('%')
	}
	sq.Args = append(sq.Args, p)
	return "like ?"
}

func (sq *SqlFunc) likeLeft(param reflect.Value) string {
	p := fmt.Sprint(param)
	lb := strings.Builder{}
	if !strings.HasPrefix(p, "%") {
		lb.WriteByte('%')
	}
	lb.WriteString(p)
	sq.Args = append(sq.Args, p)
	return "like ?"
}

func (sq *SqlFunc) marshal(list ...reflect.Value) (string, error) {
	sb := &strings.Builder{}
	for i, v := range list {
		if i > 0 {
			sb.WriteString(",")
		}
		vi := v.Interface()
		mJson, err := json.Marshal(vi)
		if err != nil {
			return "", err
		}
		sb.WriteString("?")
		sq.Args = append(sq.Args, string(mJson))
	}
	return sb.String(), nil
}

func (sq *SqlFunc) inParameter(list ...reflect.Value) string {
	sb := &strings.Builder{}
	var num int
	for _, v := range list {
		if v.Kind() == reflect.Slice {
			for i := 0; i < v.Len(); i++ {
				if num > 0 {
					sb.WriteString(",")
				}
				num++
				sb.WriteString("?")
				sq.Args = append(sq.Args, v.Index(i).Interface())
			}
		} else {
			if num > 0 {
				sb.WriteString(",")
			}
			num++
			sb.WriteString("?")
			sq.Args = append(sq.Args, v.Interface())
		}
	}
	return sb.String()
}

func (sq *SqlFunc) setParameter(list ...reflect.Value) (string, error) {
	sb := &strings.Builder{}
	preAlias := ""
	var num int
	for _, param := range list {
		switch param.Kind() {
		case reflect.String:
			if preAlias == "" {
				preAlias = param.Interface().(string) + "."
			}
		case reflect.Map:
			if param.Type().Key().Kind() != reflect.String {
				preAlias = ""
				continue
			}
			iter := param.MapRange()
			for iter.Next() {
				name := iter.Key().Interface().(string)
				if num > 0 {
					sb.WriteString(",")
				}
				num++
				sb.WriteString(preAlias)
				sb.WriteString(name)
				sb.WriteString(" = ?")
				sq.Args = append(sq.Args, iter.Value().Interface())
			}
			preAlias = ""
		case reflect.Struct:
			for i := 0; i < param.NumField(); i++ {
				val := param.Field(i).Interface()
				if truth, ok := template.IsTrue(val); ok && truth {
					name := param.Type().Field(i).Name
					if num > 0 {
						sb.WriteString(",")
					}
					num++
					sb.WriteString(preAlias)
					sb.WriteString(name)
					sb.WriteString(" = ?")
					sq.Args = append(sq.Args, val)
				}
			}
			preAlias = ""
		default:
			return "", fmt.Errorf("setParameter sql function in paramter is not string, map or struct")
		}
	}
	return sb.String(), nil
}

func (sq *SqlFunc) whereParameter(list ...reflect.Value) (string, error) {
	sb := &strings.Builder{}
	preAlias := ""
	var num int
	for _, param := range list {
		switch param.Kind() {
		case reflect.String:
			if preAlias == "" {
				preAlias = param.Interface().(string) + "."
			}
		case reflect.Map:
			if param.Type().Key().Kind() != reflect.String {
				preAlias = ""
				continue
			}
			iter := param.MapRange()
			for iter.Next() {
				name := iter.Key().Interface().(string)
				if num > 0 {
					sb.WriteString(" and ")
				}
				num++
				sb.WriteString(preAlias)
				sb.WriteString(name)
				sb.WriteString(" = ?")
				sq.Args = append(sq.Args, iter.Value().Interface())
			}
			preAlias = ""
		case reflect.Struct:
			for i := 0; i < param.NumField(); i++ {
				val := param.Field(i).Interface()
				if truth, ok := template.IsTrue(val); ok && truth {
					name := param.Type().Field(i).Name
					if num > 0 {
						sb.WriteString(" and ")
					}
					num++
					sb.WriteString(preAlias)
					sb.WriteString(name)
					sb.WriteString(" = ?")
					sq.Args = append(sq.Args, val)
				}
			}
			preAlias = ""
		default:
			return "", fmt.Errorf("setParameter sql function in paramter is not string, map or struct")
		}
	}
	return sb.String(), nil
}
