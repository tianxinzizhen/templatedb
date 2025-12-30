这个项目是一个基于Go语言的SQL模板库，它可以帮助你在Go项目中使用SQL模板来动态生成SQL语句。
原生支持Go语言的数据库操作接口，你可以直接使用它来执行SQL语句。

获取参数的方式：
- 常量：你可以使用常量来表示参数值。
- 模板操作符：你可以使用模板操作符（{.field}、{@field}、{field}）来引用参数值。
- 数据库字段名：你可以使用数据库字段名来引用它们。（例如{user_name}转换成{.UserName}）
- 占位符：你可以使用占位符（?）来表示参数值，占位符的顺序与参数值的顺序相同。

type User struct {
	ID     int64
	UserName string
	Age      int
    CreateTime time.Time
}

create table user (
    id int64 primary key,
    user_name varchar(255),
    age int,
    create_time timestamp
);

插入多字段获取时可以不写.，直接使用结构体的字段名。
例如values ({id}, {user_name}, {age})
模板操作的.符号等价于@符号
例如
insert into user (id, user_name, age)
values ({id, user_name, age}) 
等价于 
insert into user (id, user_name, age)
values ({@id, @user_name, @age}) 
等价于
insert into user (id, user_name, age)
values ({.id, .user_name, .age}) 
等价于
insert into user (id, user_name, age)
values ({.id}, {.user_name}, {.age}) 

如果有段sql时可选的，你可以使用方括号[]来包裹它。
select * from user where 1 = 1 [and id = {Id}];
等价于
select * from user where 1 = 1 {if .Id }and id = {Id} {end};

如果有多个可选的条件，你可以使用多个方括号[]来包裹它们。
select * from user where 1 = 1 [and id = {Id}] [and user_name = {UserName}];
等价于
select * from user where 1 = 1 {if .Id }and id = {Id} {end} {if .UserName }and user_name = {UserName} {end};

使用占位符号?来表示参数值，占位符的顺序与参数值的顺序相同。
例如
insert into user (id, user_name, age)
values (?, ?, ?)
等价于
insert into user (id, user_name, age)
values ({.id}, {.user_name}, {.age}) 