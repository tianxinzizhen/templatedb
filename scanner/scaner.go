package scanner

import (
	"database/sql"
	"reflect"
	"time"
	"unsafe"
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
	fv := s.Dest.FieldByIndex(s.Index)
	if !fv.CanSet() {
		fv = reflect.NewAt(fv.Type(), unsafe.Pointer(fv.UnsafeAddr())).Elem()
	}
	if s.Convert != nil {
		return s.Convert(fv, src)
	}
	return ConvertAssign(fv.Addr().Interface(), src)
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
		if vt.Kind() == reflect.Interface {
			if s.Column.ScanType().ConvertibleTo(vt) {
				dest := reflect.New(s.Column.ScanType()).Interface()
				ConvertAssign(dest, src)
				sc := scanTypeConvert(dest)
				s.Dest.SetMapIndex(reflect.ValueOf(s.Name), sc.Convert(vt))
			}
		} else {
			dest := reflect.New(vt)
			err := ConvertAssign(dest.Interface(), src)
			if err != nil {
				if s.Column.ScanType().ConvertibleTo(vt) {
					dest := reflect.New(s.Column.ScanType()).Interface()
					ConvertAssign(dest, src)
					sc := scanTypeConvert(dest)
					s.Dest.SetMapIndex(reflect.ValueOf(s.Name), sc.Convert(vt))
				} else {
					return err
				}
			} else {
				s.Dest.SetMapIndex(reflect.ValueOf(s.Name), dest.Elem())
			}
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
		dest := reflect.New(vt)
		if vt.Kind() == reflect.Interface {
			if s.Column.ScanType().ConvertibleTo(vt) {
				dest := reflect.New(s.Column.ScanType()).Interface()
				ConvertAssign(dest, src)
				sc := scanTypeConvert(dest)
				s.Dest.Index(s.Index).Set(sc.Convert(vt))
			}
		} else {
			err := ConvertAssign(dest.Interface(), src)
			if err != nil {
				if s.Column.ScanType().ConvertibleTo(vt) {
					dest := reflect.New(s.Column.ScanType()).Interface()
					ConvertAssign(dest, src)
					sc := scanTypeConvert(dest)
					s.Dest.Index(s.Index).Set(sc.Convert(vt))
				} else {
					return err
				}
			} else {
				s.Dest.Index(s.Index).Set(dest.Elem())
			}
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
		s.Dest.Set(reflect.New(s.Dest.Type()).Elem())
		if s.Convert != nil {
			return s.Convert(s.Dest, src)
		}
		return ConvertAssign(s.Dest.Addr().Interface(), src)
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
