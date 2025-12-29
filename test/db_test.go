package test

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"github.com/tianxinzizhen/tgsql"
	"github.com/tianxinzizhen/tgsql/sqlval"
)

//go:embed *
var testDbSql embed.FS

func GetDBFuncTemplateDB() (*tgsql.TgenSql, error) {
	sqldb, err := sql.Open("mysql", "lix:lix@1234@tcp(localhost:3306)/lix_test?charset=utf8mb4&parseTime=True&loc=Local")
	if err != nil {
		return nil, err
	}
	tdb := tgsql.NewTgenSql(sqldb)
	err = tdb.LoadFuncDataInfo(testDbSql)
	if err != nil {
		return nil, err
	}
	return tdb, nil
}

func TestSelect(t *testing.T) {
	sqlval.RegisterScanVal(&IdScan{})
	tdb, err := GetDBFuncTemplateDB()
	if err != nil {
		t.Error(err)
		return
	}
	db, err := NewTestDB(tdb)
	if err != nil {
		t.Error(err)
		return
	}
	list, _, err := db.Select(context.Background(), 1)
	if err != nil {
		t.Error(err)
		return
	}
	b, _ := json.Marshal(list)
	fmt.Println(string(b))
	// for _, v := range list {
	// 	fmt.Println(v)
	// }
}

func TestSelectByPoint(t *testing.T) {
	tdb, err := GetDBFuncTemplateDB()
	if err != nil {
		t.Error(err)
		return
	}
	db, err := NewTestDB(tdb)
	if err != nil {
		t.Error(err)
		return
	}
	list, err := db.SelectAtSignByTestInfo(context.Background(), &Test{
		Id:   1,
		Name: "a",
	})
	if err != nil {
		t.Error(err)
		return
	}
	for _, v := range list {
		fmt.Println(v)
	}
}

func TestInsert(t *testing.T) {
	tdb, err := GetDBFuncTemplateDB()
	if err != nil {
		t.Error(err)
		return
	}
	db, err := NewTestDB(tdb)
	if err != nil {
		t.Error(err)
		return
	}
	result, err := db.Insert(context.Background(), &Test{
		Id:   2,
		Name: "b",
	})
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(result.RowsAffected())
}

func TestUpdate(t *testing.T) {
	tdb, err := GetDBFuncTemplateDB()
	if err != nil {
		t.Error(err)
		return
	}
	db, err := NewTestDB(tdb)
	if err != nil {
		t.Error(err)
		return
	}
	err = db.UpdateNotResultId(context.Background(), &Test{
		Id:   2,
		Name: "b",
	})
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(db.SelectOneNoReturnErr(context.Background(), 2))
}

func TestInsert2(t *testing.T) {
	tdb, err := GetDBFuncTemplateDB()
	if err != nil {
		t.Error(err)
		return
	}
	db, err := NewTestDB(tdb)
	if err != nil {
		t.Error(err)
		return
	}
	result, err := db.Insert2(context.Background(), &Test2{
		Id:   2,
		Name: "b",
		Extend: Test{
			Id:   2,
			Name: "b",
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(result.RowsAffected())
}

func TestSelect2(t *testing.T) {
	tdb, err := GetDBFuncTemplateDB()
	if err != nil {
		t.Error(err)
		return
	}
	db, err := NewTestDB(tdb)
	if err != nil {
		t.Error(err)
		return
	}
	list, err := db.Select2(context.Background(), 2)
	if err != nil {
		t.Error(err)
		return
	}
	j, _ := json.Marshal(list)
	fmt.Println(string(j))
	// for _, v := range list {
	// 	fmt.Println(v)
	// }
}

func TestSelect2COne(t *testing.T) {
	tdb, err := GetDBFuncTemplateDB()
	if err != nil {
		t.Error(err)
		return
	}
	db, err := NewTestDB(tdb)
	if err != nil {
		t.Error(err)
		return
	}
	test, err := db.Select2COne(context.Background(), &Test2{
		Id: 2,
		Extend: Test{
			Id:   2,
			Name: "b",
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(test)
}
