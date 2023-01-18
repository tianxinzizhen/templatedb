package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/tianxinzizhen/templatedb"
)

type MTest struct {
	templatedb.DBFunc[MTest]
	Select            func(map[string]any, context.Context) ([]GoodShop, error)
	Exec              func([]GoodShop) (templatedb.Result, error)
	ExecNoResult      func([]GoodShop)
	ExecNoResultError func([]GoodShop) error
	PrepareExec       func([]GoodShop) templatedb.PrepareResult
}

func TestMakeSelectFunc(t *testing.T) {
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	dest := &MTest{}
	_, err = templatedb.DBFuncInit(dest, db)
	if err != nil {
		t.Error(err)
	}
	defer dest.Recover(&err)
	data, err := dest.Select(nil, context.Background())
	if err != nil {
		t.Error(err)
	}
	for _, v := range data {
		fmt.Printf("%#v\n", v)
	}
	// fmt.Printf("%#v", dest.Select(db, map[string]any{
	// 	"id": 1,
	// }))
}

func TestMakeExecFunc(t *testing.T) {
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	dest := &MTest{}
	_, err = templatedb.DBFuncInit(dest, db)
	if err != nil {
		t.Error(err)
	}
	defer db.Recover(&err)
	dest, _ = dest.Begin()
	defer dest.AutoCommit(&err)
	a, err := dest.Exec([]GoodShop{{
		Name:         "insertOne",
		UserId:       2,
		Phone:        "12345678910",
		Introduction: "一些简单的介绍1",
		Avatar:       "aa.jpg",
		Image:        "bb.jpg",
	}})
	fmt.Println(a)
	fmt.Println(err)
}

func TestMakeExecFuncNoResult(t *testing.T) {
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	dest := &MTest{}
	_, err = templatedb.DBFuncInit(dest, db)
	if err != nil {
		t.Error(err)
	}
	defer db.Recover(&err)
	dest, _ = dest.Begin()
	defer dest.AutoCommit(&err)
	dest.ExecNoResult([]GoodShop{{
		Name:         "insertOne",
		UserId:       2,
		Phone:        "12345678910",
		Introduction: "一些简单的介绍1",
		Avatar:       "aa.jpg",
		Image:        "bb.jpg",
	}})
}

func TestMakeExecFuncNoResultError(t *testing.T) {
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	dest := &MTest{}
	_, err = templatedb.DBFuncInit(dest, db)
	if err != nil {
		t.Error(err)
	}
	defer db.Recover(&err)
	dest, _ = dest.Begin()
	defer dest.AutoCommit(&err)
	err = dest.ExecNoResultError([]GoodShop{{
		Name:         "insertOne",
		UserId:       2,
		Phone:        "12345678910",
		Introduction: "一些简单的介绍1",
		Avatar:       "aa.jpg1",
		Image:        "bb.jpg",
	}})
	if err != nil {
		t.Error(err)
	}
}

func TestMakePrepareExecFunc(t *testing.T) {
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	dest := &MTest{}
	_, err = templatedb.DBFuncInit(dest, db)
	if err != nil {
		t.Error(err)
	}
	defer db.Recover(&err)
	a := dest.PrepareExec([]GoodShop{{
		Name:         "insertOne",
		UserId:       2,
		Phone:        "12345678910",
		Introduction: "一些简单的介绍",
		Avatar:       "aa.jpg",
		Image:        "bb.jpg",
	}, {
		Name:         "insertOne",
		UserId:       2,
		Phone:        "12345678910",
		Introduction: "一些简单的介绍",
		Avatar:       "aa.jpg",
		Image:        "bb.jpg",
	}})
	fmt.Println(a)
}

type TestJson struct {
	Id    int    `json:"id"`
	Jname string `json:"jname"`
	User  *User  `json:"user"`
}

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}
type MTestJson struct {
	templatedb.DBFunc[MTestJson]
	Insert func(*TestJson)
	Select func() []TestJson
}

func TestJSONInert(t *testing.T) {
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	dest, err := templatedb.DBFuncInit(&MTestJson{}, db)
	if err != nil {
		t.Error(err)
	}
	dest.Insert(&TestJson{
		Jname: "qwer",
		User: &User{
			Name: "lix",
			Age:  16,
		},
	})
}

func TestJSONSelect(t *testing.T) {
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	dest, err := templatedb.DBFuncInit(&MTestJson{}, db)
	if err != nil {
		t.Error(err)
	}
	sv := dest.Select()
	for _, v := range sv {
		fmt.Printf("%#v", v.User)
	}
}
