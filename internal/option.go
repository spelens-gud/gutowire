package internal

import (
	"path/filepath"
	"strings"
)

// opt struct    存储配置选项.
type opt struct {
	searchPath string   // 依赖搜索路径，指定在哪个目录下查找依赖
	pkg        string   // 生成文件的包名
	genPath    string   // 生成文件的输出路径
	initWire   []string // 需要生成初始化函数的类型列表
}

// Option 配置函数类型，用于设置 opt.
type Option func(*opt)

// newGenOpt function    创建并初始化配置选项
//
// genPath: 生成文件的目标路径
// opts: 可选的配置函数
func newGenOpt(genPath string, opts ...Option) *opt {
	o := &opt{
		genPath: genPath,
	}
	for _, opts := range opts {
		opts(o)
	}
	o.init()
	return o
}

// init function    初始化配置选项.
func (o *opt) init() {
	// 如果未指定包名，尝试从目录推断
	if len(o.pkg) == 0 {
		var err error
		// 尝试从现有 Go 文件中读取包名
		o.pkg, err = getPathGoPkgName(o.genPath)
		if err != nil {
			// 如果失败，使用目录名作为包名（将 - 替换为 _）
			o.pkg = strings.ReplaceAll(filepath.Base(o.genPath), "-", "_")
		}
	}
	// 如果未指定搜索路径，使用 go.mod 所在目录
	if len(o.searchPath) == 0 {
		modPath := getGoModDir()
		if len(modPath) > 0 {
			o.searchPath = modPath
		}
	}
}
