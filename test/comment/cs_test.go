package comment

import (
	"embed"
	"fmt"
	"testing"

	"github.com/tianxinzizhen/templatedb"
	"github.com/tianxinzizhen/templatedb/test"
)

//go:embed cs.go
var sqlDir embed.FS

func GetDB() (*TestDB, error) {
	db, err := test.GetOptionDB()
	if err != nil {
		return nil, err
	}
	dest := &TestDB{}
	_, err = templatedb.DBFuncInitAndLoad(db, dest, sqlDir, templatedb.LoadComment)
	if err != nil {
		return nil, err
	}
	return dest, nil
}
func TestSelect(t *testing.T) {
	db, _ := GetDB()
	ret := db.Select()
	fmt.Printf("%#v", ret)
}
