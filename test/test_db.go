package test

import (
	"context"
	"database/sql"

	"github.com/tianxinzizhen/tgsql"
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
	// 需要返回新插入的自增id
	/*sql
	{if}insert into test values ({ id , name})
	*/
	Insert func(ctx context.Context, testInfo *Test) (sql.Result, error)

	//sql select * from test where 1 [and id=@id]
	Select func(ctx context.Context, id int) (IdScan, string, error)

	//sql select * from test2 where id=?
	Select2 func(ctx context.Context, id int) ([]Test2, error)

	//sql?option{not_prepare:true} select extend,name from test2 where id=?
	Select2COne func(ctx context.Context, id *Test2) (*Test, error)

	//sql select * from test where id=?
	SelectNoReturnErr func(ctx context.Context, id int) []*Test

	/*sql
	select * from test where id=? limit 1
	*/
	SelectOne func(ctx context.Context, id int) (*Test, error)

	/*sql
	select * from test where id=?
	*/
	SelectOneNoReturnErr func(ctx context.Context, id int) *Test

	/*sql
	select * from test where id=@id and name=@name
	*/
	SelectByTestInfo func(ctx context.Context, testInfo *Test) ([]*Test, error)

	// 使用参数符号取参数
	/*sql
	select * from test where id=@id and name=@name
	*/
	SelectAtSignByTestInfo func(ctx context.Context, testInfo *Test) ([]*Test, error)

	/*sql
	insert into test2 values(@id,@name,@extend)
	*/
	Insert2 func(ctx context.Context, testInfo *Test2) (sql.Result, error)

	/*sql
	insert into test2 (id,name,extend) values(?,?,?)
	*/
	Insert3 func(ctx context.Context, testInfo *Test2) (sql.Result, error)

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

func NewTestDB(tdb *tgsql.TgenSql) (*TestDB, error) {
	ret := &TestDB{}
	err := tgsql.InitDBFunc(tdb, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
