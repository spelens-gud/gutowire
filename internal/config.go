package internal

import (
	"text/template"
)

var (
	// wireTag 注解标记，用于标识需要进行依赖注入的类型或函数.
	wireTag = "@autowire"
	// filePrefix 生成文件的前缀名称.
	filePrefix = "autowire"
)

// setTemp 预编译的 Set 模板，用于快速生成代码.
var setTemp = template.Must(template.New("").Parse(setTemplate))

// element 表示一个可注入的组件(结构体或函数).
type element struct {
	name        string   // 组件名称，如 Zoo、Cat
	constructor string   // 构造函数名称，如 NewZoo、InitCat
	fields      []string // 结构体字段列表（用于 config 模式）
	implements  []string // 实现的接口列表
	pkg         string   // 所在包名
	pkgPath     string   // 完整的包导入路径
	initWire    bool     // 是否标记为 @autowire.init
	configWire  bool     // 是否标记为 @autowire.config
}

// wireSet struct    表示一个 Wire Set 的配置信息.
type wireSet struct {
	Package string   // 包名
	Items   []string // Set 中包含的所有项（构造函数、结构体等）
	SetName string   // Set 的名称，如 AnimalsSet
}

// WithPkg function    设置生成文件的包名
// 如果不设置，会自动从目录名推断.
func WithPkg(pkg string) Option {
	return func(o *opt) {
		o.pkg = pkg
	}
}

// InitStruct function    指定需要生成初始化函数的结构体类型
// 参数为空或 "*" 表示为所有标记 @autowire.init 的类型生成初始化函数
// 示例: InitStruct("Zoo", "MiniZoo") 只为 Zoo 和 MiniZoo 生成初始化函数.
func InitStruct(initStruct ...string) Option {
	return func(o *opt) {
		if len(initStruct) == 0 {
			initStruct = []string{"*"}
		}
		o.initWire = initStruct
	}
}

// WithSearchPath function    设置依赖搜索路径
// 指定在哪个目录下递归查找带 @autowire 注解的代码
// 如果不设置，默认使用 go.mod 所在目录.
func WithSearchPath(path string) Option {
	return func(o *opt) {
		o.searchPath = path
	}
}
