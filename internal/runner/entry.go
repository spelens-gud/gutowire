// Package runner 提供了 gutowire 的主执行流程。
// 协调配置初始化、代码扫描、Wire 配置生成和 wire 命令执行等步骤。
package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spelens-gud/gutowire/internal/config"
	"github.com/spelens-gud/gutowire/internal/errors"
	"github.com/spelens-gud/gutowire/internal/generator"
	"github.com/spelens-gud/gutowire/internal/parser"
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
func RunAutoWire(genPath string, opts ...config.Option) error {
	// 第一步：生成 Wire 配置文件
	if err := runAutoWireGen(genPath, opts...); err != nil {
		return fmt.Errorf("生成 Wire 配置文件失败: %w", err)
	}

	log.Printf("Wire 配置文件写入成功")

	// 第二步：调用 wire 命令生成最终代码
	if err := runWire(genPath); err != nil {
		// 使用友好的错误提示
		if wireErr, ok := err.(*errors.FriendlyError); ok {
			return wireErr
		}
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
func runAutoWireGen(genPath string, opts ...config.Option) error {
	// 初始化配置选项
	o := config.NewGenOpt(genPath, opts...)
	file := o.SearchPath
	pkg := strings.ReplaceAll(o.Pkg, "-", "_") // 包名中的 - 替换为 _（Go 包名规范）

	// 获取模块基础路径
	modBase, err := parser.GetModBase()
	if err != nil {
		return fmt.Errorf("获取模块基础路径失败: %w", err)
	}

	// 创建搜索器实例
	sc := generator.NewAutoWireSearcher(genPath, modBase, o.InitWire, pkg)

	// 扫描所有文件，收集注解信息
	if err := sc.SearchAllPath(file); err != nil {
		return fmt.Errorf("扫描文件失败: %w", err)
	}
	log.Printf("autowire 注解分析完成")

	// 如果没有找到任何注解，直接返回
	if len(sc.ElementMap) == 0 {
		log.Printf("未找到任何 @autowire 注解")
		return nil
	}

	// 生成 Wire 配置文件
	if err := sc.Write(); err != nil {
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
		return &errors.FriendlyError{
			Type:    errors.ErrorTypeFileNotFound,
			Message: "未找到 wire 命令",
			Suggestions: []string{
				"运行以下命令安装 wire: go install github.com/google/wire/cmd/wire@latest",
				"确保 $GOPATH/bin 或 $GOBIN 在 PATH 环境变量中",
				"检查 Go 环境是否正确配置",
			},
			HelpURL: "https://github.com/google/wire#installation",
		}
	}

	// 检查是否为可信的 bin 目录
	if !strings.Contains(wirePath, "bin") {
		return fmt.Errorf("wire 命令路径不安全: %s", wirePath)
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 在指定目录下执行 wire 命令
	//nolint:gosec
	cmd := exec.CommandContext(ctx, wirePath)
	cmd.Dir = path
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[生成失败] %s", output)
		// 返回友好的错误提示
		return errors.NewWireError(string(output))
	}
	log.Printf("[生成成功] %s", output)
	return nil
}

// formatWireError function    格式化 wire 错误信息.
func formatWireError(output string) string {
	if output == "" {
		return "未知错误"
	}

	var friendlyMsg strings.Builder
	friendlyMsg.WriteString("Wire 依赖注入生成失败，请检查以下问题：\n\n")

	lines := strings.Split(output, "\n")
	errorCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析 wire 错误信息
		if strings.Contains(line, "provider struct has multiple fields of type invalid type") {
			errorCount++
			// 提取文件路径
			if idx := strings.Index(line, ":"); idx > 0 {
				filePath := line[strings.Index(line, " ")+1 : idx]
				friendlyMsg.WriteString(fmt.Sprintf("x 错误 %d: 结构体字段类型错误\n", errorCount))
				friendlyMsg.WriteString(fmt.Sprintf("   文件: %s\n", filePath))
				friendlyMsg.WriteString("   原因: 结构体中存在多个相同类型的匿名字段，或字段类型无法解析\n")
				friendlyMsg.WriteString("   建议:\n")
				friendlyMsg.WriteString("   - 检查是否有重复的匿名字段（embedded fields）\n")
				friendlyMsg.WriteString("   - 确保所有字段类型都已正确导入\n")
				friendlyMsg.WriteString("   - 避免循环依赖\n\n")
			}
		} else if strings.Contains(line, "generate failed") {
			errorCount++
			friendlyMsg.WriteString(fmt.Sprintf("x 错误 %d: %s\n\n", errorCount, line))
		} else if strings.HasPrefix(line, "wire:") {
			// 保留原始 wire 错误信息作为详细信息
			if !strings.Contains(line, "provider struct has multiple fields") {
				friendlyMsg.WriteString(fmt.Sprintf("   详细: %s\n", strings.TrimPrefix(line, "wire:")))
			}
		}
	}

	if errorCount == 0 {
		friendlyMsg.WriteString("原始错误信息:\n")
		friendlyMsg.WriteString(output)
	}

	return friendlyMsg.String()
}
