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

func TestSelect(t *testing.T) {
	db, err := test.GetOptionDB()
	if err != nil {
		t.Error(err)
	}
	dest := &TestAA{}
	_, err = templatedb.DBFuncInitAndLoad(db, dest, sqlDir, templatedb.LoadComment)
	if err != nil {
		t.Error(err)
	}
	ret := dest.Select()
	fmt.Printf("%#v", ret)
}
