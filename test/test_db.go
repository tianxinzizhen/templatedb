package test

import (
	"context"
	"database/sql"
	"embed"

	"github.com/tianxinzizhen/templatedb"
)

/*
init sql :
create table test(

	id int,
	name varchar(20)

);
insert into test values(1,"a");
*/
type TestDB struct {
	/*sql
	select * from test where id=? limit 1
	*/
	SelectOne func(ctx context.Context, id int) (*Test, error)

	//sql select * from test where [id=@id]
	Select func(ctx context.Context, id int) (IdScan, string, error)

	//sql select * from test2 where id=?
	Select2 func(ctx context.Context, id int) ([]Test2, error)

	//sql?option{not_prepare:true} select extend,name from test2 where id=?
	Select2COne func(ctx context.Context, id *Test2) (*Test, error)

	//sql select * from test where id=?
	SelectNoReturnErr func(ctx context.Context, id int) []*Test

	/*sql
	select * from test where id=?
	*/
	SelectOneNoReturnErr func(ctx context.Context, id int) *Test

	// ----parameter is point

	// 如果比较符在前面是字段名称,那么默认取该名称参数,否则按参数顺序取
	/*sql
	select * from test where id=@id and name=@name
	*/
	SelectByTestInfo func(ctx context.Context, testInfo *Test) ([]*Test, error)

	// 使用参数符号取参数
	/*sql
	select * from test where id=@id and name=@name
	*/
	SelectAtSignByTestInfo func(ctx context.Context, testInfo *Test) ([]*Test, error)

	// 需要返回新插入的自增id
	/*sql
	insert into test values({id,name})
	*/
	Insert func(ctx context.Context, testInfo *Test) (sql.Result, error)

	/*sql
	insert into test2 values(@id,@name,@:extend)
	*/
	Insert2 func(ctx context.Context, testInfo *Test2) (sql.Result, error)

	/*sql
	insert into test values(@id,@name)
	*/
	InsertNotResultId func(ctx context.Context, testInfo *Test) error

	// 需要返回新插入的受影响id
	/*sql
	update test
	set name=@name
	where id=@id
	*/
	Update func(ctx context.Context, testInfo *Test) (sql.Result, error)

	/*sql?option{not_prepare:true}
	update test
	set name=@name
	where id=@id
	*/
	UpdateNotResultId func(ctx context.Context, testInfo *Test) error
}

//go:embed test_db.go
var testDbSql embed.FS

func NewTestDB(tdb *templatedb.DBFuncTemplateDB) (*TestDB, error) {
	ret := &TestDB{}
	err := templatedb.DBFuncContextInit(tdb, ret, templatedb.LoadComment, testDbSql)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
