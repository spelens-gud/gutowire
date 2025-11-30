package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// FileConfig struct    配置文件结构.
type FileConfig struct {
	// 基础配置
	SearchPath  string   `yaml:"search_path"`  // 依赖搜索路径
	OutputPath  string   `yaml:"output_path"`  // 输出路径
	Package     string   `yaml:"package"`      // 包名
	InitTypes   []string `yaml:"init_types"`   // 需要生成初始化函数的类型
	EnableCache bool     `yaml:"enable_cache"` // 是否启用缓存
	Parallel    int      `yaml:"parallel"`     // 并发数，0 表示自动
	ExcludeDirs []string `yaml:"exclude_dirs"` // 排除的目录
	IncludeOnly []string `yaml:"include_only"` // 只包含的目录
	Watch       bool     `yaml:"watch"`        // 是否启用 watch 模式
	WatchIgnore []string `yaml:"watch_ignore"` // watch 模式忽略的文件模式
}

// DefaultConfig function    返回默认配置.
func DefaultConfig() *FileConfig {
	return &FileConfig{
		EnableCache: true,
		Parallel:    0, // 自动检测
		ExcludeDirs: []string{"vendor", "testdata", ".git"},
		Watch:       false,
	}
}

// LoadConfigFile function    从文件加载配置.
func LoadConfigFile(path string) (*FileConfig, error) {
	// 如果路径为空，尝试查找默认配置文件
	if path == "" {
		path = findConfigFile()
		if path == "" {
			return DefaultConfig(), nil
		}
	}

	//nolint:gosec
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return cfg, nil
}

// SaveConfigFile method    保存配置到文件.
func (c *FileConfig) SaveConfigFile(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	//nolint:gosec
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// findConfigFile function    查找配置文件.
func findConfigFile() string {
	// 按优先级查找配置文件
	candidates := []string{
		".gutowire.yaml",
		".gutowire.yml",
		"gutowire.yaml",
		"gutowire.yml",
	}

	for _, name := range candidates {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}

	return ""
}

// ToOptions method    将配置文件转换为选项.
func (c *FileConfig) ToOptions() []Option {
	var opts []Option

	if c.Package != "" {
		opts = append(opts, WithPkg(c.Package))
	}

	if c.SearchPath != "" {
		opts = append(opts, WithSearchPath(c.SearchPath))
	}

	if len(c.InitTypes) > 0 {
		opts = append(opts, InitStruct(c.InitTypes...))
	}

	return opts
}

// GenerateExampleConfig function    生成示例配置文件.
func GenerateExampleConfig(path string) error {
	example := &FileConfig{
		SearchPath:  "./",
		OutputPath:  "./wire",
		Package:     "wire",
		InitTypes:   []string{"App", "Server"},
		EnableCache: true,
		Parallel:    0,
		ExcludeDirs: []string{"vendor", "testdata", ".git"},
		Watch:       false,
		WatchIgnore: []string{"*.gen.go", "wire_gen.go"},
	}

	return example.SaveConfigFile(path)
}
