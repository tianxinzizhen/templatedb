# tgsql - Go语言SQL模板库

`tgsql` 是一个强大的Go语言SQL模板库，它允许您使用模板动态生成SQL语句，并原生支持Go语言的数据库操作接口，您可以直接使用它来执行SQL语句。

## 特性

- ✅ 动态SQL生成：使用模板语法动态生成SQL语句
- ✅ 原生数据库支持：直接兼容Go标准库的`database/sql`接口
- ✅ 多种参数引用方式：支持常量、模板操作符和占位符
- ✅ 丰富的模板函数：提供like、in、set、where等实用函数
- ✅ 可选条件支持：使用方括号[]轻松处理可选SQL条件
- ✅ JSON支持：内置JSON序列化函数
- ✅ 自定义模板函数：支持扩展自定义模板函数
- ✅ SQL日志：内置SQL执行日志功能

## 安装

```bash
go get github.com/tianxinzizhen/tgsql
```

## 快速开始

### 初始化

```go
import (
    "database/sql"
    "github.com/tianxinzizhen/tgsql"
    _ "github.com/go-sql-driver/mysql"
)

func main() {
    // 连接数据库
    db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // 创建TgenSql实例
    tdb := tgsql.NewTgenSql(db)
}
```

### 参数引用方式

- **常量**：直接使用常量值
- **模板操作符**：`{.field}`、`{@field}`、`{field}`
- **数据库字段名**：自动转换为结构体字段（如`{user_name}` → `{.UserName}`）
- **占位符**：使用`?`作为参数占位符

## 模板语法

### 基本语法

```go
type User struct {
    ID         int64
    UserName   string
    Age        int
    CreateTime time.Time
}
```

```sql
-- SQL表结构
CREATE TABLE user (
    id int64 primary key,
    user_name varchar(255),
    age int,
    create_time timestamp
);
```

### 字段引用

```sql
-- 以下方式等价
INSERT INTO user (id, user_name, age)
VALUES ({id}, {user_name}, {age});

INSERT INTO user (id, user_name, age)
VALUES ({@id, @user_name, @age});

INSERT INTO user (id, user_name, age)
VALUES ({.id, .user_name, .age});

INSERT INTO user (id, user_name, age)
VALUES ({.id}, {.user_name}, {.age});
```

### 可选条件

使用方括号`[]`包裹可选的SQL条件：

```sql
SELECT * FROM user WHERE 1 = 1 [AND id = {Id}] [AND user_name = {UserName}];

-- 等价于
SELECT * FROM user WHERE 1 = 1 {if .Id} AND id = {Id} {end} {if .UserName} AND user_name = {UserName} {end};
```

### 实用模板函数

#### 1. Like 函数

```sql
-- 不同匹配模式
SELECT * FROM user WHERE user_name {like .user_name};  -- %value%
SELECT * FROM user WHERE user_name {liker .user_name}; -- %value
SELECT * FROM user WHERE user_name {likel .user_name}; -- value%
```

#### 2. Param 函数

```sql
-- 简化多参数输出
INSERT INTO user (id, user_name, age)
VALUES ({param .id .user_name .age});

-- 等价于
INSERT INTO user (id, user_name, age)
VALUES ({.id}, {.user_name}, {.age});
```

#### 3. JSON 序列化

```sql
-- 将对象转换为JSON字符串
INSERT INTO user (id, info)
VALUES ({.id}, {json .info});

-- 或
INSERT INTO user (id, info)
VALUES ({.id}, {marshal .info});
```

#### 4. In 函数

```sql
-- 处理单个值
SELECT * FROM user WHERE id {in .id};

-- 处理数组
SELECT * FROM user WHERE id {in .ids};
```

#### 5. Set 函数

自动生成UPDATE语句的SET部分：

```sql
UPDATE user SET {set .user} WHERE id = {.id};

-- 如果user是结构体，会自动过滤无效值
-- 等价于：id = {.ID}, user_name = {.UserName}, age = {.Age}, create_time = {.CreateTime}

-- 使用表别名
UPDATE user u SET {set "u" .user} WHERE u.id = {.id};
```

#### 6. Where 函数

自动生成WHERE条件：

```sql
SELECT * FROM user WHERE {where .user};

-- 如果user是结构体，会自动过滤无效值
-- 等价于：id = {.ID} AND user_name = {.UserName} AND age = {.Age} AND create_time = {.CreateTime}

-- 使用表别名
SELECT * FROM user u WHERE {where "u" .user};
```

## 完整示例

### 示例程序
main.go
```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/tianxinzizhen/tgsql"
    _ "github.com/go-sql-driver/mysql"
)

// User 用户模型
type User struct {
    ID         int64     `json:"id"`
    UserName   string    `json:"user_name"`
    Age        int       `json:"age"`
    Email      string    `json:"email"`
    CreateTime time.Time `json:"create_time"`
}

func main() {
    // 连接数据库
    db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // 测试数据库连接
    if err := db.Ping(); err != nil {
        panic(err)
    }
    fmt.Println("数据库连接成功")

    // 创建TgenSql实例
    tdb := tgsql.NewTgenSql(db)

    // 设置SQL日志
    tdb.SqlLogFunc(func(ctx context.Context, funcName, sql string, args ...any) {
        fmt.Printf("[%s] %s: %s %v\n", time.Now().Format("2006-01-02 15:04:05"), funcName, sql, args)
    })

    // 创建用户表（仅示例）
    createTableSQL := `CREATE TABLE IF NOT EXISTS user (
        id int64 PRIMARY KEY AUTO_INCREMENT,
        user_name varchar(255) NOT NULL,
        age int,
        email varchar(255),
        create_time timestamp DEFAULT CURRENT_TIMESTAMP
    )`
    _, err = db.Exec(createTableSQL)
    if err != nil {
        panic(err)
    }
    fmt.Println("用户表创建成功")
```

## 核心概念

`tgsql` 使用结构体方法签名结合SQL模板的方式来执行SQL语句。您需要：

1. 定义一个包含方法签名的结构体,且注释是//sql 或者/*sql 开头的
2. 使用`InitDBFunc`初始化结构体方法

### 1. 定义接口结构体

user_db.go
```go
// UserDB 用户数据访问接口, sql模板数据通过注释写入
type UserDB struct {
    *tgsql.TgenSql // 嵌入TgenSql

    /*sql
    INSERT INTO user (user_name, age, email) VALUES ({.UserName}, {.Age}, {.Email});
    */
    Insert   func(ctx context.Context, user *User) (sql.Result, error)
    /*sql
    SELECT id, user_name, age, email, create_time FROM user WHERE id = {.id};
    */
    GetByID  func(ctx context.Context, id int64) (*User, error)
    /*sql
    UPDATE user SET {set .} WHERE id = {.ID};
    */
    Update   func(ctx context.Context, user *User) error
    /*sql
    SELECT id, user_name, age, email, create_time FROM user 
    WHERE 1=1 [AND age > {.age}] [AND user_name {like .keyword}] 
    ORDER BY id;
    */
    List     func(ctx context.Context, age int, keyword string) ([]*User, error)
    /*sql
    DELETE FROM user WHERE id = {.id};
    */
    Delete   func(ctx context.Context, id int64) error
}
```

#### 2. 加载模板并初始化

```go
//go:embed user_db.go
var userSql string
// 加载SQL模板
if err := tdb.LoadFuncDataInfoString(userSql); err != nil {
    panic(err)
}

// 初始化UserDB
var userDB UserDB
if err := tdb.InitDBFunc(&userDB); err != nil {
    panic(err)
}

// 调用示例

// 插入用户
newUser := &User{
    UserName: "张三",
    Age:      25,
    Email:    "zhangsan@example.com",
}
result, err := userDB.Insert(context.Background(), newUser)
if err != nil {
    panic(err)
}
lastID, _ := result.LastInsertId()
fmt.Printf("插入用户成功，ID: %d\n", lastID)

// 查询用户
user, err := userDB.GetByID(context.Background(), lastID)
if err != nil {
    panic(err)
}
fmt.Printf("查询用户成功: %+v\n", user)

// 更新用户
user.Age = 26
user.Email = "zhangsan-updated@example.com"
if err := userDB.Update(context.Background(), user); err != nil {
    panic(err)
}
fmt.Println("更新用户成功")

// 列表查询
users, err := userDB.List(context.Background(), 20, "张")
if err != nil {
    panic(err)
}
fmt.Printf("条件查询结果: %+v\n", users)

// 删除用户
if err := userDB.Delete(context.Background(), lastID); err != nil {
    panic(err)
}
fmt.Printf("删除用户成功，ID: %d\n", lastID)
```

## 模板文件加载方式

`tgsql` 支持多种方式加载SQL模板：

```go
// 1. 从字符串加载
//go:embed user_db.go
var userSql string
if err := tdb.LoadFuncDataInfoString(userSql); err != nil {
    panic(err)
}

// 2. 从字节数组加载
userSqlBytes := []byte(userSql)
if err := tdb.LoadFuncDataInfoBytes(userSqlBytes); err != nil {
    panic(err)
}

// 3. 从embed.FS加载 (推荐用于生产环境)
//go:embed *
var sqlFiles embed.FS
if err := tdb.LoadFuncDataInfo(sqlFiles); err != nil {
    panic(err)
}
```

## 高级特性

### 自定义模板函数

```go
// 添加自定义模板函数
tdb.AddTemplateFunc("customFunc", func(arg string) string {
    return "custom_" + arg
})

// 使用自定义函数
insertSQL := `INSERT INTO user (user_name) VALUES ({customFunc .UserName})`
```

### SQL日志

```go
// 设置SQL执行日志
tdb.SqlLogFunc(func(ctx context.Context, funcName, sql string, args ...any) {
    log.Printf("SQL执行: %s %v", sql, args)
})
```

### 自定义分隔符

```go
// 修改模板分隔符（默认是{}}
tdb.Delims("{{", "}}")
```

### 使用sql字符串替换模板变量

```go
// 使用sql字符串替换模板变量
    /*sql
    SELECT id, user_name, age, email, create_time FROM user_{lang} 
    WHERE 1=1 [AND age > {.age}] [AND user_name {like .keyword}] 
    ORDER BY id;
    */
    List     func(ctx context.Context, lang sqlwrite.Sql, age int, keyword string) ([]*User, error)
    /*sql
    INSERT INTO user_{lang} (user_name) VALUES ({.UserName})
    */
    Insert   func(ctx context.Context, lang sqlwrite.Sql, user *User) (sql.Result, error)
```
调用示例

```go
// 列表查询
users, err := userDB.List(context.Background(), sqlwrite.Sql("en"), 20, "张")
if err != nil {
    panic(err)
}
fmt.Printf("条件查询结果: %+v\n", users)

// 插入用户
newUser := &User{
    UserName: "张三",
}
result, err := userDB.Insert(context.Background(), sqlwrite.Sql("en"), newUser)
if err != nil {
    panic(err)
}
lastID, _ := result.LastInsertId()
fmt.Printf("插入用户成功，ID: %d\n", lastID)
```

### 注册参数转换器

```go
// 必须实现该接口
// type Convert[T any] interface {
// 	ConvertValue(v T) (any, error)
// 	ConvertValuePtr(v *T) (any, error)
// }

// 如果参数是一个特殊的时间类型，例如time.Time
type Time2 struct {
    time.Time
}

type Time2Converter struct{}
// 实现Convert接口
func (t *Time2Converter) ConvertValue(v Time2) (any, error) {
    return v.Time, nil
}
func (t *Time2Converter) ConvertValuePtr(v *Time2) (any, error) {
    return v.Time, nil
}

// 注册参数转换器
tdb.RegisterParamConverter(&Time2Converter{})
```

### 注册结果扫描器

```go
// 必须实现该接口
// type Scan[T any] interface {
// 	ScanValue(v any) (T, error)
// 	ScanValuePtr(v any) (*T, error)
// }

// 如果结果是一个特殊的时间类型，例如time.Time
type Time2 struct {
    time.Time
}

type Time2Scanner struct{
    t time.Time
}

// 实现Scan接口
func (ts *Time2Scanner) Scan(dest any) error {
	if t, ok := dest.(time.Time); ok {
		ts.t = t
	}
	return nil
}

func (t *Time2Scanner) ScanValue() (Time2, error) {
    return Time2{Time: t.t}, nil
}

func (t *Time2Scanner) ScanValuePtr() (*Time2, error) {
    return &Time2{Time: t.t}, nil
}

// 注册结果扫描器
tdb.RegisterResultScanner(&Time2Scanner{})
```


## 测试

项目包含完整的测试用例，您可以通过以下命令运行：

```bash
go test ./test
```

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request！
