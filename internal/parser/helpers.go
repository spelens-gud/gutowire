package parser

import (
	"bytes"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/imports"
)

var (
	// modTmp 缓存 go.mod 文件路径.
	modTmp string
	// o 确保 go.mod 路径只查询一次.
	o sync.Once
	// importOpt goimports 的格式化选项.
	importOpt = &imports.Options{
		Comments:  true,
		TabIndent: true,
		TabWidth:  8,
	}
	// importMu 保护 import 处理过程的并发安全.
	importMu sync.Mutex
)

// GetPathGoPkgName    获取指定目录的 Go 包名
// 通过解析目录中的 .go 文件来确定包名.
func GetPathGoPkgName(pathStr string) (pkg string, err error) {
	entries, err := os.ReadDir(pathStr)
	if err != nil {
		return "", fmt.Errorf("读取目录失败: %w", err)
	}

	// 如果目录为空，使用目录名作为包名
	if len(entries) == 0 {
		return getGoPkgNameByDir(pathStr), nil
	}

	for _, entry := range entries {
		// 跳过目录
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// 跳过非 Go 文件
		if !CheckFileType(name) {
			continue
		}

		// 解析 Go 文件获取包名
		//nolint:gosec
		bs, err := os.ReadFile(filepath.Join(pathStr, name))
		if err != nil {
			return "", fmt.Errorf("读取文件 %s 失败: %w", name, err)
		}

		// 解析文件
		f, err := parser.ParseFile(token.NewFileSet(), "", bs, parser.ParseComments)
		if err != nil {
			return "", fmt.Errorf("解析文件 %s 失败: %w", name, err)
		}

		return f.Name.Name, nil
	}

	return "", errors.New("目录中未找到有效的 Go 源文件")
}

// getGoPkgNameByDir    使用目录名作为包名
// 这是一个后备方案，当无法从文件中读取包名时使用.
func getGoPkgNameByDir(pathStr string) (pkg string) {
	return filepath.Base(pathStr)
}

// CheckFileType function    检查文件类型.
// 用于跳过非go文件或者测试文件.
func CheckFileType(name string) bool {
	if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
		return false
	}
	return true
}

// GetGoModDir 获取 go.mod 文件所在的目录
// 这通常是项目的根目录.
func GetGoModDir() (modPath string) {
	mod := GetGoModFilePath()
	modPath = filepath.Dir(mod)
	return
}

// GetGoModFilePath    获取 go.mod 文件的完整路径
// 使用 sync.Once 确保只执行一次 go env 命令.
func GetGoModFilePath() (modPath string) {
	o.Do(func() {
		// 执行 go env GOMOD 获取 go.mod 路径
		cmd := exec.Command(
			"go",
			"env",
			"GOMOD",
		)
		stdout := &bytes.Buffer{}
		cmd.Stdout = stdout
		_ = cmd.Run()
		modTmp = strings.Trim(stdout.String(), "\n")
	})
	return modTmp
}

// GetModBase function    获取当前 Go 模块的基础路径
// 例如: github.com/Just-maple/go-autowire
// 这个路径用于计算包的完整导入路径.
func GetModBase() (modBase string, err error) {
	modPath := GetGoModFilePath()
	//nolint:gosec
	mb, err := os.ReadFile(modPath)
	if err != nil {
		return "", fmt.Errorf("读取 go.mod 文件失败: %w", err)
	}

	// 解析 go.mod 文件
	f, err := modfile.Parse("", mb, nil)
	if err != nil {
		return "", fmt.Errorf("解析 go.mod 文件失败: %w", err)
	}

	// 提取 module 声明的路径
	if f.Module == nil {
		return "", errors.New("go.mod 文件中缺少 module 声明，请检查 go 环境配置")
	}
	modBase = f.Module.Mod.Path
	return
}

// GetPkgPath function    计算文件的完整包导入路径
// 例如: github.com/Just-maple/go-autowire/example/dependencies
//
// filePath: 文件的绝对或相对路径
// modBase: 模块的基础路径.
func GetPkgPath(filePath, modBase string) (pkgPath string) {
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return
	}

	dir := GetGoModDir()
	if len(abs) < len(dir) {
		return
	}

	// 计算相对于模块根目录的路径，并拼接模块基础路径
	pkgPath = filepath.ToSlash(filepath.Dir(filepath.Join(modBase, abs[len(dir):])))
	return
}

// AppendPkg function    拼接包名和选择器
// 如果包名为空，直接返回选择器
// 例如: appendPkg("pkg", "Type") -> "pkg.Type".
func AppendPkg(pkg string, sel string) string {
	if len(pkg) == 0 {
		return sel
	}
	return pkg + "." + sel
}

// ImportAndWrite function    自动添加缺失的 import，移除未使用的 import，并格式化代码.
func ImportAndWrite(filename string, src []byte) error {
	writeData, err := importProcess(src)
	if err != nil {
		return fmt.Errorf("处理 import 语句失败: %w", err)
	}
	// 写入文件
	//nolint:gosec
	if err := os.WriteFile(filename, writeData, 0644); err != nil {
		return fmt.Errorf("写入文件 %s 失败: %w", filename, err)
	}
	return nil
}

// importProcess function    处理代码的 import 语句
// 使用 goimports 自动添加、删除和格式化 import.
func importProcess(src []byte) ([]byte, error) {
	importMu.Lock()
	defer importMu.Unlock()

	result, err := imports.Process("", src, importOpt)
	if err != nil {
		return nil, fmt.Errorf("goimports 处理失败: %w", err)
	}
	return result, nil
}
