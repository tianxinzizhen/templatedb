package test

import (
	"database/sql"
	"embed"
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/tianxinzizhen/templatedb"
)

//go:embed sql/*
var sqlDir embed.FS

type GoodShop struct {
	Id           int    `json:"id"`
	UserId       int    `json:"userId"`
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	Introduction string `json:"introduction"`
	Avatar       string `json:"avatar"`
	Image        string `json:"image"`
	Status       int    `json:"status"`
}

func getDB() (*templatedb.DefaultDB, error) {
	sqldb, err := sql.Open("mysql", "lix:lix@/test")
	if err != nil {
		return nil, err
	}
	return templatedb.NewDefaultDB(sqldb, templatedb.LoadSqlOfXml(sqlDir))
}

func TestGetDb(t *testing.T) {
	_, err := getDB()
	if err != nil {
		t.Error(err)
	}
}

var testParam = []struct {
	name  string
	param any
}{
	{name: "select", param: GoodShop{
		Name:  "0店铺1",
		Phone: "12345678910",
	}},
	{name: "selectAtsign", param: nil},
	{name: "sqlparam", param: GoodShop{
		Name: "0店铺1",
	}},
	{name: "sqlparam", param: GoodShop{
		Name: "0店铺1",
	}},
	{name: "", param: GoodShop{
		Name: "3店铺1",
	}},
}

func TestSelect(t *testing.T) {
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	for _, tp := range testParam[4:5] {
		ret, err := templatedb.DBSelect[GoodShop](db).Select(tp.param, tp.name)
		if err != nil {
			t.Error(err)
		}
		for _, v := range ret {
			fmt.Printf("%#v\n", v)
		}
	}
}

var TestInFunctionParams = []struct {
	name  string
	param map[string]any
}{
	{name: "inints", param: map[string]any{"ids": []int{1, 3, 10}}},
	{name: "inStructs", param: map[string]any{"ids": []GoodShop{{Id: 1}, {Id: 3}, {Id: 10}}}},
	{name: "inMaps", param: map[string]any{"ids": []map[string]any{
		{"id": 1},
		{"id": 3},
		{"id": 10},
	}}},
}

func TestInFunction(t *testing.T) {
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	for _, tp := range TestInFunctionParams {
		ret, err := templatedb.DBSelect[GoodShop](db).Select(tp.param, tp.name)
		if err != nil {
			t.Error(err)
		}
		for _, v := range ret {
			fmt.Printf("%#v\n", v)
		}
	}
}

var TestInsertParams = []struct {
	name  string
	param any
}{
	{name: "insertOne", param: GoodShop{
		Name:         "insertOne",
		UserId:       2,
		Phone:        "12345678910",
		Introduction: "一些简单的介绍",
		Avatar:       "aa.jpg",
		Image:        "bb.jpg",
	}},
	{name: "insertList", param: []GoodShop{{
		Name:         "insertList1",
		UserId:       2,
		Phone:        "12345678910",
		Introduction: "一些简单的介绍",
		Avatar:       "aa.jpg",
		Image:        "bb.jpg",
	}, {
		Name:         "insertList2",
		UserId:       2,
		Phone:        "12345678910",
		Introduction: "一些简单的介绍",
		Avatar:       "aa.jpg",
		Image:        "bb.jpg",
	},
	}},
	{name: "insertListParam", param: []GoodShop{{
		Name:         "insertListParam1",
		UserId:       2,
		Phone:        "12345678910",
		Introduction: "一些简单的介绍",
		Avatar:       "aa.jpg",
		Image:        "bb.jpg",
	}, {
		Name:         "insertListParam2",
		UserId:       2,
		Phone:        "12345678910",
		Introduction: "一些简单的介绍",
		Avatar:       "aa.jpg",
		Image:        "bb.jpg",
	},
	}},
}

func TestInsert(t *testing.T) {
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	for _, tp := range TestInsertParams {
		lastInsertId, rowsAffected, err := db.Exec(tp.param, tp.name)
		if err != nil {
			t.Error(err)
		}
		fmt.Printf("lastInsertId:%d,rowsAffected:%d\n", lastInsertId, rowsAffected)
	}
}

func TestInsertTx(t *testing.T) {
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	for _, tp := range TestInsertParams {
		var txfunc = func() {
			tx, err := db.Begin()
			defer tx.AutoCommit(&err)
			lastInsertId, rowsAffected, err := tx.Exec(tp.param, TestInsert, tp.name)
			if err != nil {
				t.Error(err)
			}
			fmt.Printf("lastInsertId:%d,rowsAffected:%d\n", lastInsertId, rowsAffected)
		}
		txfunc()
	}
}
