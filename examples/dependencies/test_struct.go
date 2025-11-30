package dependencies

import (
	"github.com/spelens-gud/gutowire/examples/dependencies/test_b"
	"github.com/spelens-gud/gutowire/examples/dependencies/test_b/test"
	"github.com/spelens-gud/gutowire/examples/dependencies/test_c"
)

// @autowire.init(set=struct)
type Test struct {
	T4     test.Test
	Test2_ Test2       // 给匿名字段添加名称，避免类型冲突
	Test3_ Test3       // 给匿名字段添加名称，避免类型冲突
	Test4_ Test4       // 给匿名字段添加名称，避免类型冲突
	TestC  test_c.Test // 给匿名字段添加名称，避免类型冲突
	T1     test_b.Test
	T3     test_b.Test2
	T2     test_c.Test2
	T      TestInterface1
}

type TestInterface1 interface {
}

// @autowire(set=struct,TestInterface1)
type Test2 struct{ Test3 }

// @autowire(set=struct,new=ConstTest3)
type Test3 struct{}

func ConstTest3() Test3 {
	return Test3{}
}

// @autowire(set=struct)
type Test4 struct{}

func NewTest4() Test4 {
	return Test4{}
}

// @autowire(set=func)
func UselessFunc() interface{} {
	return nil
}
