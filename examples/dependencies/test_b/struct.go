// Package test_b 提供了测试用的依赖组件。
// 用于演示跨包的依赖注入和包名冲突处理。
package test_b

import (
	test2 "github.com/spelens-gud/gutowire/examples/dependencies/test_b/test"
	test_b2 "github.com/spelens-gud/gutowire/examples/dependencies/test_b/test/test_b"
)

// @autowire(set=struct)
type Test struct {
	Test2
	test2.Test
}

// @autowire(set=struct)
type Test2 struct {
	test_b2.Test
}
