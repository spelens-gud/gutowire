// Package config 提供了 gutowire 的配置管理功能。
// 包含配置选项的定义和处理，支持自定义包名、搜索路径、初始化类型等配置。
package config

var (
	// WireTag 注解标记，用于标识需要进行依赖注入的类型或函数.
	WireTag = "@autowire"
	// FilePrefix 生成文件的前缀名称.
	FilePrefix = "autowire"
)

// WithPkg function    设置生成文件的包名
// 如果不设置，会自动从目录名推断.
func WithPkg(pkg string) Option {
	return func(o *Opt) {
		o.Pkg = pkg
	}
}

// InitStruct function    指定需要生成初始化函数的结构体类型
// 参数为空或 "*" 表示为所有标记 @autowire.init 的类型生成初始化函数
// 示例: InitStruct("Zoo", "MiniZoo") 只为 Zoo 和 MiniZoo 生成初始化函数.
func InitStruct(initStruct ...string) Option {
	return func(o *Opt) {
		if len(initStruct) == 0 {
			initStruct = []string{"*"}
		}
		o.InitWire = initStruct
	}
}

// WithSearchPath function    设置依赖搜索路径
// 指定在哪个目录下递归查找带 @autowire 注解的代码
// 如果不设置，默认使用 go.mod 所在目录.
func WithSearchPath(path string) Option {
	return func(o *Opt) {
		o.SearchPath = path
	}
}
