// Package test_c 提供了另一组测试用的依赖组件。
// 用于演示多包依赖和依赖链的处理。
package test_c

import (
	test_b2 "github.com/spelens-gud/gutowire/examples/dependencies/test_b"
)

// @autowire(set=struct)
type Test struct {
	Test2
}

// @autowire(set=struct)
type Test2 struct {
	T3 test_b2.Test2
}
