package scan

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"strings"
)

type ScanValJson struct {
	Val reflect.Value
}

func isScanValJson(columns *sql.ColumnType) bool {
	return strings.ToLower(columns.DatabaseTypeName()) == "json"
}

func ShouldScanValJson(columns *sql.ColumnType, val reflect.Value) reflect.Value {
	if isScanValJson(columns) {
		scanVal := reflect.New(reflect.TypeOf(ScanValJson{}))
		if s, ok := scanVal.Interface().(*ScanValJson); ok {
			s.Val = val
		}
		return scanVal
	}
	return val
}

func (s *ScanValJson) Scan(src any) error {
	if src == nil {
		return nil
	}
	if srcBytes, ok := src.([]byte); ok {
		if len(srcBytes) == 0 {
			return nil
		}
		if s.Val.Kind() == reflect.Pointer {
			if err := json.Unmarshal(srcBytes, s.Val.Interface()); err != nil {
				return err
			}
		} else {
			if s.Val.CanAddr() {
				if err := json.Unmarshal(srcBytes, s.Val.Addr().Interface()); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return nil
}

func getScanValJson(val reflect.Value) reflect.Value {
	if val.Type() == reflect.TypeFor[*ScanValJson]() {
		return val.Interface().(*ScanValJson).Val
	}
	return val
}
