package comment

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"reflect"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/tianxinzizhen/templatedb"
)

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

type MTest struct {
	templatedb.DBFunc[MTest]
	//select * from tbl_test
	Select func(map[string]any, context.Context) ([]GoodShop, error)
	/*
		INSERT INTO tbl_test
		  (userId, name, phone, introduction, avatar, image, status)
		  VALUES {range $i,$v:=. } {comma $i}
		  (@userId, @name, @phone, @introduction, @avatar, @image, @status)
		  {end}
	*/
	Exec func([]GoodShop) (templatedb.Result, error)
	/*
			INSERT INTO tbl_test
		        (userId, name, phone, introduction, avatar, image, status)
		        VALUES {range $i,$v:=. } {comma $i}
		        (@userId, @name, @phone, @introduction, @avatar, @image, @status)
		        {end}
	*/
	ExecNoResult func([]GoodShop)
	/*
			INSERT INTO tbl_test
		        (userId, name, phone, introduction, avatar, image, status)
		        VALUES {range $i,$v:=. } {comma $i}
		        (@userId, @name, @phone, @introduction, @avatar, @image, @status)
		        {end}
	*/
	ExecNoResultError func([]GoodShop) error
	/*
		INSERT INTO tbl_test (userId, name, phone, introduction, avatar, image, status) VALUES
		    (@userId, @name, @phone, @introduction, @avatar, @image, @status)
	*/
	PrepareExec func([]GoodShop) templatedb.PrepareResult
}

var pkg = reflect.TypeOf((*MTest)(nil)).Elem().PkgPath()

//go:embed *
var sqlDir embed.FS

func getDB() (*templatedb.DefaultDB, error) {
	sqldb, err := sql.Open("mysql", "root:lz@3306!@tcp(mysql.local.lezhichuyou.com:3306)/lz_tour_lix?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true")
	if err != nil {
		return nil, err
	}
	templatedb.RecoverPrintf = fmt.Printf
	return templatedb.NewDefaultDB(sqldb, templatedb.LoadSqlOfCommentStruct(pkg, sqlDir))
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
