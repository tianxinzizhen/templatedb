package templatedb

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/tianxinzizhen/templatedb/sqlwrite"
	"github.com/tianxinzizhen/templatedb/template"

	"github.com/tianxinzizhen/templatedb/util"
)

var sqlFunc template.FuncMap = make(template.FuncMap)

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
