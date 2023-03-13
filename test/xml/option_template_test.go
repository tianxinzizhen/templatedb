package xml

import (
	"embed"
	"fmt"
	"testing"

	"github.com/tianxinzizhen/templatedb"
	"github.com/tianxinzizhen/templatedb/test"
)

//go:embed sql
var sqlDir embed.FS

func GetOptionDB() (*templatedb.OptionDB, error) {
	tdb, err := test.GetOptionDB()
	if err != nil {
		return nil, err
	}
	tdb.LoadSqlOfXml(sqlDir)
	return tdb, nil
}
func TestOptionSelectScan(t *testing.T) {
	db, err := GetOptionDB()
	if err != nil {
		t.Error(err)
	}
	db.Query(&templatedb.ExecOption{
		Sql: "select UserId, Name FROM tbl_test",
		Result: func(id int, name string) {
			fmt.Println(id, name)
		},
	})
}

type Info struct {
	UserId int
	Name   string
}

func TestOptionSelect(t *testing.T) {
	db, err := GetOptionDB()
	if err != nil {
		t.Error(err)
	}
	//查询多条时使用的行来判断不同的sql语句
	ret := db.Query(&templatedb.ExecOption{
		Sql:    "select UserId, Name FROM tbl_test",
		Result: []*Info{},
	}).([]*Info)
	for _, v := range ret {
		fmt.Println(v)
	}
	ret = db.Query(&templatedb.ExecOption{
		Sql:    "select Name FROM tbl_test",
		Result: []*Info{},
	}).([]*Info)
	for _, v := range ret {
		fmt.Println(v)
	}
}

func TestOptionSelectArgs(t *testing.T) {
	db, err := GetOptionDB()
	if err != nil {
		t.Error(err)
	}
	//查询多条时使用的行来判断不同的sql语句
	ret := db.Query(&templatedb.ExecOption{
		Sql:    "select UserId, Name FROM tbl_test where Name={param .Name} and  UserId=? ",
		Result: []*Info{},
		Args:   []any{1},
		Param:  Info{Name: "dd"},
	}).([]*Info)
	for _, v := range ret {
		fmt.Println(v)
	}
}

func TestOptionSelectXml(t *testing.T) {
	db, err := GetOptionDB()
	if err != nil {
		t.Error(err)
	}
	//查询多条时使用的行来判断不同的sql语句
	ret := db.Query(&templatedb.ExecOption{
		FuncPC: templatedb.FuncPC(TestOptionSelectXml),
		Result: []*Info{},
		Args:   []any{1},
		Param:  Info{Name: "dd"},
	}).([]*Info)
	for _, v := range ret {
		fmt.Println(v)
	}
}
