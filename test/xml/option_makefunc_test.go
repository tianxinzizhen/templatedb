package xml

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/tianxinzizhen/templatedb"
)

type GoodShop struct {
	Name string
}
type OptionMTest struct {
	templatedb.DBFunc[OptionMTest]
	Select            func(map[string]any, context.Context) ([]OptionTblTest, error)
	Exec              func([]GoodShop) (templatedb.Result, error)
	ExecNoResult      func([]GoodShop)
	ExecNoResultError func([]GoodShop) error
}

type OptionTblTest struct {
	Id         int
	UserId     int
	Name       string
	created_at time.Time
}

func TestOptionMakeSelectFunc(t *testing.T) {
	db, err := GetOptionDB()
	if err != nil {
		t.Error(err)
	}
	dest := &OptionMTest{}
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
}
