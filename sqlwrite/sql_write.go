package sqlwrite

import (
	"strings"
)

type SqlWrite struct {
	sql  strings.Builder
	args []any
}

func (s *SqlWrite) Write(p []byte) (n int, err error) {
	n, err = s.sql.Write(p)
	return
}

func (s *SqlWrite) Sql() string {
	return s.sql.String()
}

func (s *SqlWrite) Args() []any {
	return s.args
}

func (s *SqlWrite) WriteString(str string) (n int, err error) {
	n, err = s.sql.WriteString(str)
	return
}

func (s *SqlWrite) WriteParam(sql string, arg any) {
	if arg == nil {
		return
	}
	if sqw, ok := arg.(*SqlWrite); ok {
		s.args = append(s.args, sqw.Args()...)
		s.sql.WriteString(sqw.Sql())
		return
	} else {
		s.sql.WriteString(sql)
		s.args = append(s.args, arg)
	}
}
