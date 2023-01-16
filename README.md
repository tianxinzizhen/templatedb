# 简介
使用sql模版和参数动态生成sql,并且参数化执行,可以用模版提供更复杂的sql操作

# 注意
* 默认使用参数解析符号是：@ 同时支持使用模版函数{param .args} 进行参数化提取
* 默认使用结构字段别名tag是：json
更改字段别名函数: template.TagAsFieldName = JsonTagAsFieldName

# 错误接收
* 如果要得到templatedb的错误信息需要在代码开头使用  
defer db.Recover(&err)  $~~~~$//注意：这里是输出错误参数的引用，传入其他错误对象将不能得到引用指针
* 事务开启后有通过错误自动提交的方法  
defer tx.AutoCommit(&err) $~~~~$//注意：同样是输出错误参数的引用



# 如何使用
查看文件test/sql/test_sql.xml,test/templatedb_test.go
有几个例子看，前置需要对golang的模板熟悉。

# 安全相关
[![Security Status](https://www.murphysec.com/platform3/v3/badge/1612004657648414720.svg?t=1)](https://www.murphysec.com/accept?code=decf9bb2d4c69750e880241c395edbd7&type=1&from=2&t=2)