package sqlwrite

import "strings"

type SqlWrite struct {
	Sql  strings.Builder
	Args []any
}

func (s *SqlWrite) Write(p []byte) (n int, err error) {
	n, err = s.Sql.Write(p)
	return
}

func (s *SqlWrite) String() string {
	return s.Sql.String()
}

func (s *SqlWrite) WriteString(str string) (n int, err error) {
	n, err = s.Sql.WriteString(str)
	return
}

func (s *SqlWrite) AddArgs(arg any) {
	if arg == nil {
		return
	}
	if sqw, ok := arg.(*SqlWrite); ok {
		s.Args = append(s.Args, sqw.Args...)
		s.Sql.WriteString(sqw.String())
		return
	} else {
		s.Args = append(s.Args, arg)
	}
}
