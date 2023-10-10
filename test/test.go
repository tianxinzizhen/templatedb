package test

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/tianxinzizhen/templatedb"
)

func GetOptionDB() (*templatedb.DBFuncTemplateDB, error) {
	sqldb, err := sql.Open("mysql", "root:lz@3306!@tcp(mysql.local.lezhichuyou.com:3306)/lz_tour?charset=utf8mb4&parseTime=True&loc=Local")
	if err != nil {
		return nil, err
	}
	return templatedb.NewDBFuncTemplateDB(sqldb), nil
}
