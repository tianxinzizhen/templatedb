package scanner

import (
	"database/sql"
	"reflect"
	"time"
)

type StructScanner struct {
	Dest    reflect.Value
	Convert func(field reflect.Value, v any) error
	Index   []int
}

func (s *StructScanner) Scan(src any) error {
	if src == nil {
		return nil
	}
	if s.Convert != nil {
		return s.Convert(s.Dest.FieldByIndex(s.Index), src)
	}
	return ConvertAssign(s.Dest.FieldByIndex(s.Index).Addr().Interface(), src)
}

type MapScanner struct {
	Dest   reflect.Value
	Column *sql.ColumnType
	Name   string
}

func (s *MapScanner) Scan(src any) error {
	if src == nil {
		return nil
	} else {
		vt := s.Dest.Type().Elem()
		dest := reflect.New(s.Column.ScanType()).Interface()
		ConvertAssign(dest, src)
		sc := scanTypeConvert(dest)
		if sc.CanConvert(vt) {
			s.Dest.SetMapIndex(reflect.ValueOf(s.Name), sc.Convert(vt))
		}
		return nil
	}
}

type SliceScanner struct {
	Dest   reflect.Value
	Column *sql.ColumnType
	Index  int
}

func (s *SliceScanner) Scan(src any) error {
	if src == nil {
		return nil
	} else {
		vt := s.Dest.Type().Elem()
		dest := reflect.New(s.Column.ScanType()).Interface()
		ConvertAssign(dest, src)
		sc := scanTypeConvert(dest)
		if sc.CanConvert(vt) {
			s.Dest.Index(s.Index).Set(sc.Convert(vt))
		}
		return nil
	}
}

type ParameterScanner struct {
	Dest    reflect.Value
	Column  *sql.ColumnType
	Convert func(field reflect.Value, v any) error
}

func (s *ParameterScanner) Scan(src any) error {
	if src == nil {
		return nil
	} else {
		vt := s.Dest.Type()
		dest := reflect.New(s.Column.ScanType()).Interface()
		ConvertAssign(dest, src)
		sc := scanTypeConvert(dest)
		if s.Convert != nil {
			return s.Convert(s.Dest, src)
		}
		if sc.CanConvert(vt) {
			s.Dest.Set(sc.Convert(vt))
		}
		return nil
	}
}

func scanTypeConvert(scanVal any) reflect.Value {
	var ret any
	switch v := scanVal.(type) {
	case *sql.NullBool:
		if v.Valid {
			ret = v.Bool
		} else {
			ret = false
		}
	case *sql.NullByte:
		if v.Valid {
			ret = v.Byte
		} else {
			ret = 0
		}
	case *sql.NullFloat64:
		if v.Valid {
			ret = v.Float64
		} else {
			ret = float64(0)
		}
	case *sql.NullInt16:
		if v.Valid {
			ret = v.Int16
		} else {
			ret = int16(0)
		}
	case *sql.NullInt32:
		if v.Valid {
			ret = v.Int32
		} else {
			ret = int32(0)
		}
	case *sql.NullInt64:
		if v.Valid {
			ret = v.Int64
		} else {
			ret = int64(0)
		}
	case *sql.NullString:
		if v.Valid {
			ret = v.String
		} else {
			ret = ""
		}
	case *sql.NullTime:
		if v.Valid {
			ret = v.Time
		} else {
			ret = time.Time{}
		}
	case *sql.RawBytes:
		ret = string(*v)
	default:
		ret = reflect.ValueOf(v).Elem().Interface()
	}
	return reflect.ValueOf(ret)
}
