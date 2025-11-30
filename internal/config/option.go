package config

import (
	"path/filepath"
	"strings"

	"github.com/spelens-gud/gutowire/internal/parser"
)

// Opt struct    存储配置选项.
type Opt struct {
	SearchPath string   // 依赖搜索路径，指定在哪个目录下查找依赖
	Pkg        string   // 生成文件的包名
	GenPath    string   // 生成文件的输出路径
	InitWire   []string // 需要生成初始化函数的类型列表
}

// Option 配置函数类型，用于设置 Opt.
type Option func(*Opt)

// NewGenOpt function    创建并初始化配置选项
//
// genPath: 生成文件的目标路径
// opts: 可选的配置函数
func NewGenOpt(genPath string, opts ...Option) *Opt {
	o := &Opt{
		GenPath: genPath,
	}
	for _, opt := range opts {
		opt(o)
	}
	o.init()
	return o
}

// init function    初始化配置选项.
func (o *Opt) init() {
	// 如果未指定包名，尝试从目录推断
	if len(o.Pkg) == 0 {
		var err error
		// 尝试从现有 Go 文件中读取包名
		o.Pkg, err = parser.GetPathGoPkgName(o.GenPath)
		if err != nil {
			// 如果失败，使用目录名作为包名（将 - 替换为 _）
			o.Pkg = strings.ReplaceAll(filepath.Base(o.GenPath), "-", "_")
		}
	}
	// 如果未指定搜索路径，使用 go.mod 所在目录
	if len(o.SearchPath) == 0 {
		modPath := parser.GetGoModDir()
		if len(modPath) > 0 {
			o.SearchPath = modPath
		}
	}
}
