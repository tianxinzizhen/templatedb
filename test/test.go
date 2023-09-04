package test

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/tianxinzizhen/templatedb"
)

func GetOptionDB() (*templatedb.DBFuncTemplateDB, error) {
	sqldb, err := sql.Open("mysql", "root:lz@3306!@tcp(mysql.local.lezhichuyou.com:3306)/lz_tour?charset=utf8mb4&parseTime=True&loc=Local")
	if err != nil {
		return nil, err
	}
	templatedb.LogPrintf = func(_ context.Context, info string) {
		fmt.Println(info)
	}
	return templatedb.NewDBFuncTemplateDB(sqldb), nil
}
