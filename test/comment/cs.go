package comment

import (
	"github.com/tianxinzizhen/templatedb"
)

type TestAA struct {
	templatedb.DBFunc[TestAA]
	//sql select * from tbl_hotel limit 1
	Select func() map[string]any
}
type Hotel struct {
	Name string `json:"name"`
}
