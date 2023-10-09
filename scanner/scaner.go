package scanner

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"time"
	"unsafe"
)

//go:linkname convertAssign database/sql.convertAssign
func convertAssign(dest, src any) error

func notBasicType(field reflect.Type) bool {
	if field.Kind() == reflect.Pointer {
		field = field.Elem()
	}
	if field.Kind() == reflect.Struct || field.Kind() == reflect.Slice || field.Kind() == reflect.Map {
		return true
	} else {
		return false
	}
}
func jsonConvertStruct(field reflect.Value, src any) error {
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
		return convertAssign(field.Addr().Interface(), src)
	}
}

type StructScanner struct {
	Dest         reflect.Value
	Index        []int
	SetParameter func(src any) (any, error)
}

func (s *StructScanner) Scan(src any) error {
	if src == nil {
		return nil
	}
	fv := s.Dest.FieldByIndex(s.Index)
	if !fv.CanSet() {
		fv = reflect.NewAt(fv.Type(), unsafe.Pointer(fv.UnsafeAddr())).Elem()
	}
	if notBasicType(fv.Type()) {
		if s.SetParameter != nil {
			dest, err := s.SetParameter(src)
			if err != nil {
				return err
			}
			fv.Set(reflect.ValueOf(dest))
			return nil
		} else {
			if _, ok := src.(time.Time); !ok {
				return jsonConvertStruct(fv, src)
			}
		}
	}
	return convertAssign(fv.Addr().Interface(), src)
}

type ParameterScanner struct {
	Dest         reflect.Value
	Column       *sql.ColumnType
	SetParameter func(src any) (any, error)
}

func (s *ParameterScanner) Scan(src any) error {
	if src == nil {
		return nil
	} else {
		if notBasicType(s.Dest.Type()) {
			if s.SetParameter != nil {
				dest, err := s.SetParameter(src)
				if err != nil {
					return err
				}
				s.Dest.Set(reflect.ValueOf(dest))
				return nil
			} else {
				if _, ok := src.(time.Time); !ok {
					return jsonConvertStruct(s.Dest, src)
				}
			}
		}
		return convertAssign(s.Dest.Addr().Interface(), src)
	}
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
