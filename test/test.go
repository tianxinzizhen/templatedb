package test

import (
	"database/sql"
	"fmt"

	"github.com/tianxinzizhen/templatedb"
)

func GetOptionDB() (*templatedb.OptionDB, error) {
	sqldb, err := sql.Open("mysql", "root:lz@3306!@tcp(mysql.local.lezhichuyou.com:3306)/lz_tour?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true")
	if err != nil {
		return nil, err
	}
	templatedb.RecoverPrintf = fmt.Printf
	return templatedb.NewOptionDB(sqldb), nil
}
