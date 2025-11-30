// Package errors 提供友好的错误提示和建议
package errors

import (
	"fmt"
	"strings"
)

// ErrorType 错误类型.
type ErrorType int

const (
	// ErrorTypeUnknown 未知错误.
	ErrorTypeUnknown ErrorType = iota
	// ErrorTypeCircularDep 循环依赖.
	ErrorTypeCircularDep
	// ErrorTypeMissingDep 缺少依赖.
	ErrorTypeMissingDep
	// ErrorTypeInvalidAnnotation 无效注解.
	ErrorTypeInvalidAnnotation
	// ErrorTypeWireError Wire 错误.
	ErrorTypeWireError
	// ErrorTypeFileNotFound 文件未找到.
	ErrorTypeFileNotFound
)

// FriendlyError struct    友好的错误信息.
type FriendlyError struct {
	Type        ErrorType // 错误类型
	Message     string    // 错误信息
	Suggestions []string  // 建议列表
	Details     string    // 错误详情
	HelpURL     string    // 帮助链接
}

// Error method    实现 error 接口.
func (e *FriendlyError) Error() string {
	var sb strings.Builder

	sb.WriteString("x ")
	sb.WriteString(e.Message)
	sb.WriteString("\n\n")

	if e.Details != "" {
		sb.WriteString("详细信息:\n")
		sb.WriteString(e.Details)
		sb.WriteString("\n\n")
	}

	if len(e.Suggestions) > 0 {
		sb.WriteString("! 建议:\n")
		for i, suggestion := range e.Suggestions {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, suggestion))
		}
		sb.WriteString("\n")
	}

	if e.HelpURL != "" {
		sb.WriteString(fmt.Sprintf("> 更多帮助: %s\n", e.HelpURL))
	}

	return sb.String()
}

// NewCircularDepError function    创建循环依赖错误.
func NewCircularDepError(pkg string) *FriendlyError {
	return &FriendlyError{
		Type:    ErrorTypeCircularDep,
		Message: fmt.Sprintf("检测到循环依赖: 包 %s 已导入生成目标包", pkg),
		Suggestions: []string{
			"将生成的代码移到单独的包中",
			"检查是否有不必要的导入",
			"使用接口来解耦依赖关系",
		},
		HelpURL: "https://github.com/spelens-gud/gutowire#circular-dependency",
	}
}

// NewMissingDepError function    创建缺少依赖错误.
func NewMissingDepError(typeName string) *FriendlyError {
	return &FriendlyError{
		Type:    ErrorTypeMissingDep,
		Message: fmt.Sprintf("无法找到类型 %s 的依赖", typeName),
		Suggestions: []string{
			"确保所有依赖都已添加 @autowire 注解",
			"检查包导入路径是否正确",
			"确认类型名称拼写正确",
		},
		HelpURL: "https://github.com/spelens-gud/gutowire#missing-dependency",
	}
}

// NewInvalidAnnotationError function    创建无效注解错误.
func NewInvalidAnnotationError(annotation string, reason string) *FriendlyError {
	return &FriendlyError{
		Type:    ErrorTypeInvalidAnnotation,
		Message: "无效的注解: " + annotation,
		Details: reason,
		Suggestions: []string{
			"检查注解语法是否正确",
			"参考文档中的注解示例",
			"确保括号和参数格式正确",
		},
		HelpURL: "https://github.com/spelens-gud/gutowire#annotation-syntax",
	}
}

// NewWireError function    创建 Wire 错误.
func NewWireError(output string) *FriendlyError {
	suggestions := []string{
		"检查是否有循环依赖",
		"确保所有依赖都已正确注入",
		"查看上面的详细错误信息",
	}

	// 根据错误输出添加特定建议
	if strings.Contains(output, "multiple fields of type") {
		suggestions = append(suggestions, "避免在结构体中使用多个相同类型的匿名字段")
	}
	if strings.Contains(output, "no provider found") {
		suggestions = append(suggestions, "确保缺失的类型已添加 @autowire 注解")
	}

	return &FriendlyError{
		Type:        ErrorTypeWireError,
		Message:     "Wire 依赖注入生成失败",
		Details:     output,
		Suggestions: suggestions,
		HelpURL:     "https://github.com/google/wire/blob/main/docs/guide.md",
	}
}

// NewFileNotFoundError function    创建文件未找到错误.
func NewFileNotFoundError(path string) *FriendlyError {
	return &FriendlyError{
		Type:    ErrorTypeFileNotFound,
		Message: "文件或目录不存在: " + path,
		Suggestions: []string{
			"检查路径是否正确",
			"确保文件或目录存在",
			"使用绝对路径或相对于当前目录的路径",
		},
	}
}

// WrapError function    包装错误为友好错误.
func WrapError(err error, message string) *FriendlyError {
	return &FriendlyError{
		Type:    ErrorTypeUnknown,
		Message: message,
		Details: err.Error(),
		Suggestions: []string{
			"查看详细错误信息",
			"检查日志获取更多信息",
		},
	}
}
