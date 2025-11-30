package test_b

import (
	test2 "github.com/spelens-gud/gutowire/examples/dependencies/test_b/test"
)

// @autowire(set=struct)
type Test struct {
	Test_  test2.Test  // 给匿名字段添加名称，避免类型冲突
	Test2_ test2.Test2 // 给匿名字段添加名称，避免类型冲突
	T2     Test2
}

// @autowire(set=struct)
type Test2 struct {
	Test2_ test2.Test2 // 给匿名字段添加名称，避免类型冲突
}
