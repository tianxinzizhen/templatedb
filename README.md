# Introduction
SQL is dynamically generated using SQL templates and parameters, which can provide more complex SQL operations with templates


# How to use
* Table Info
```go
type Test struct{
    UserId int
    Name string
}
```
* INIT TDB
```go
	sqldb, err := sql.Open("mysql", "root:lz@3306!@tcp(localhost:3306)/lz_tour?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true")
	if err != nil {
		return nil, err
	}
	templatedb.LogPrintf = fmt.Printf  //设置错误接收打印信息
    tdb := templatedb.NewOptionDB(sqldb)
```
* SELECT LIST
```go
    //接收错误
    defer db.Recover(ctx, &err) 
	list:=tdb.TQuery(&templatedb.ExecOption{
		Sql: "select UserId, Name FROM tbl_test where UserId=? and Name=@Name",
        Args:[]any{1},
        Param:map[string]any{"Name":"test"},
		Result: []*Test{},
	}).([]*Test)
```
* SELECT ONE
```go
    //接收错误
    defer db.Recover(ctx, &err) 
	t:=tdb.TQuery(&templatedb.ExecOption{
		Sql: `select UserId, Name FROM tbl_test where UserId=?
        {if .Name}
         and Name=@Name
        {end}
        `,
        Args:[]any{1},
        Param:&Test{Name:"test"},
		Result: &Test{},
	}).(*Test)
```
* SCAN FUNC
```go
    //接收错误
    defer db.Recover(ctx, &err) 
	tdb.TQuery(&templatedb.ExecOption{
		Sql: `select UserId, Name FROM tbl_test where UserId=? 
        {if .Name}
         and Name={param .Name}
        {end}`,
        Args:[]any{1},
        Param:Test{Name:"test"},
		Result: func(id int, name string) {
			fmt.Println(id, name)
		},
	})
```
defer db.Recover(ctx, &err) 只需要在代码头部调用一次便可以捕获错误信息
* Begin/BeginTX
```go
    //tx, err := tdb.BeginTx(ctx, opts)
    tx, err := tdb.Begin()
	if err != nil {
		return nil, err
	}
    //用于错误接收和事务自动提交 该函数调用后就不用再次调用 defer db.Recover(ctx, &err)
    defer tx.AutoCommit(ctx, &err) 
```

# 安全相关
[![Security Status](https://www.murphysec.com/platform3/v3/badge/1612004657648414720.svg?t=1)](https://www.murphysec.com/accept?code=decf9bb2d4c69750e880241c395edbd7&type=1&from=2&t=2)