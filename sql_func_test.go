package tgsql

import (
	"reflect"
	"testing"
)

func TestLike(t *testing.T) {
	like := like(reflect.ValueOf("test"))
	println(like.Sql())
	println(like.Args()[0].(string))
}
