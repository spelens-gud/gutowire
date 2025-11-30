package internal

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// init function    初始化日志配置.
func init() {
	log.SetPrefix("[gutowire] ") // 设置日志前缀
	log.SetFlags(0)              // 不显示时间戳
	log.SetOutput(os.Stdout)     // 输出到标准输出
}

// RunAutoWire function    执行完整的自动装配流程
// 这是主入口函数，完成两个步骤：
// 1. 扫描注解并生成 Wire 配置文件（autowire_*.go）
// 2. 调用 wire 命令生成最终的依赖注入代码（wire_gen.go）
//
// genPath: 生成文件的目标目录
// opts: 可选配置，如搜索路径、包名等
func RunAutoWire(genPath string, opts ...Option) error {
	// 第一步：生成 Wire 配置文件
	if err := runAutoWireGen(genPath, opts...); err != nil {
		return fmt.Errorf("生成 Wire 配置文件失败: %w", err)
	}

	log.Printf("Wire 配置文件写入成功")

	// 第二步：调用 wire 命令生成最终代码
	if err := runWire(genPath); err != nil {
		return fmt.Errorf("运行 wire 命令失败: %w", err)
	}
	return nil
}

// runAutoWireGen function    执行自动装配代码生成
// 这是代码生成的核心函数，完成以下步骤：
// 1. 初始化配置
// 2. 扫描指定目录下的所有 Go 文件
// 3. 解析 @autowire 注解
// 4. 生成 Wire 配置文件
//
// genPath: 生成文件的目标目录
// opts: 可选配置
func runAutoWireGen(genPath string, opts ...Option) error {
	// 初始化配置选项
	o := newGenOpt(genPath, opts...)
	file := o.searchPath
	pkg := strings.ReplaceAll(o.pkg, "-", "_") // 包名中的 - 替换为 _（Go 包名规范）

	// 获取模块基础路径
	modBase, err := getModBase()
	if err != nil {
		return fmt.Errorf("获取模块基础路径失败: %w", err)
	}

	// 创建搜索器实例
	sc := newAutoWireSearcher(genPath, modBase, o.initWire, pkg)

	// 扫描所有文件，收集注解信息
	if err := sc.SearchAllPath(file); err != nil {
		return fmt.Errorf("扫描文件失败: %w", err)
	}
	log.Printf("autowire 注解分析完成")

	// 如果没有找到任何注解，直接返回
	if len(sc.elementMap) == 0 {
		log.Printf("未找到任何 @autowire 注解")
		return nil
	}

	// 生成 Wire 配置文件
	if err := sc.write(); err != nil {
		return fmt.Errorf("写入 Wire 配置文件失败: %w", err)
	}
	return nil
}

// runWire function    执行 Google Wire 命令行工具
// 读取生成的 autowire_*.go 文件，生成最终的 wire_gen.go.
func runWire(path string) error {
	log.Printf("开始运行 wire 命令")

	// 查找 wire 命令的路径
	wirePath, err := exec.LookPath("wire")
	if err != nil {
		return fmt.Errorf("未找到 wire 命令: %w\n请通过以下命令安装: go install github.com/google/wire/cmd/wire@latest", err)
	}

	// 在指定目录下执行 wire 命令
	cmd := exec.Command(wirePath)
	cmd.Dir = path
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[生成失败] %s", output)
		return fmt.Errorf("wire 命令执行失败: %w\n输出: %s", err, output)
	}
	log.Printf("[生成成功] %s", output)
	return nil
}
