package test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"github.com/tianxinzizhen/templatedb"
)

func GetDBFuncTemplateDB() (*templatedb.DBFuncTemplateDB, error) {
	sqldb, err := sql.Open("mysql", "root:lz@3306!@tcp(mysql.local.lezhichuyou.com:3306)/lz_tour?charset=utf8mb4&parseTime=True&loc=Local")
	if err != nil {
		return nil, err
	}
	return templatedb.NewDBFuncTemplateDB(sqldb), nil
}

func TestSelect(t *testing.T) {
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
	list, err := db.Select(context.Background(), 1)
	if err != nil {
		t.Error(err)
		return
	}
	for _, v := range list {
		fmt.Println(v)
	}
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
	fmt.Println(result)
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
