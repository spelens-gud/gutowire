// Package cmd 实现了 gutowire 的命令行接口。
// 提供了主命令和相关的子命令，处理命令行参数解析和执行流程。
package cmd

/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/x/exp/charmtone"
	"github.com/charmbracelet/x/term"
	"github.com/spelens-gud/gutowire/internal/config"
	"github.com/spelens-gud/gutowire/internal/runner"
	"github.com/spelens-gud/gutowire/internal/version"
	"github.com/spelens-gud/gutowire/internal/watcher"
	"github.com/spf13/cobra"
)

const (
	commandName = "gutowire"
)

var (
	wirePath   string
	scope      string
	pkg        string
	configFile string
	watch      bool
	noCache    bool
	initConfig bool
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   commandName,
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
	RunE: func(cmd *cobra.Command, args []string) error {
		// 如果是初始化配置文件
		if initConfig {
			return handleInitConfig()
		}

		// 加载配置文件
		cfg, err := config.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("加载配置文件失败: %w", err)
		}

		// 构建配置选项（命令行参数优先级高于配置文件）
		var opts []config.Option

		// 应用包名配置
		if pkg != "" {
			opts = append(opts, config.WithPkg(pkg))
		} else if cfg.Package != "" {
			opts = append(opts, config.WithPkg(cfg.Package))
		}

		// 应用搜索路径配置
		searchPath := scope
		if searchPath == "" && cfg.SearchPath != "" {
			searchPath = cfg.SearchPath
		}
		if searchPath != "" {
			opts = append(opts, config.WithSearchPath(searchPath))
		}

		// 从位置参数或标志或配置文件获取生成路径
		if wirePath == "" && len(args) > 0 {
			wirePath = args[0]
		}
		if wirePath == "" && cfg.OutputPath != "" {
			wirePath = cfg.OutputPath
		}

		// 验证必需参数
		if wirePath == "" {
			return fmt.Errorf("必须指定 Wire 配置文件生成路径\n使用方式: %s [flags] <生成路径>", commandName)
		}

		// 添加初始化配置
		if len(cfg.InitTypes) > 0 {
			opts = append(opts, config.InitStruct(cfg.InitTypes...))
		} else {
			opts = append(opts, config.InitStruct())
		}

		// Watch 模式
		if watch || cfg.Watch {
			return handleWatch(wirePath, searchPath, opts)
		}

		// 执行自动装配
		if err := runner.RunAutoWire(wirePath, opts...); err != nil {
			return fmt.Errorf("自动装配失败: %w", err)
		}

		fmt.Println("✓ Wire 配置文件生成成功")
		return nil
	},
}

var versionBit = lipgloss.NewStyle().Foreground(charmtone.Coral).SetString(`
  ___  _  _  ____  __   _  _  __  ____  ____ 
 / __)/ )( \(_  _)/  \ / )( \(  )(  _ \(  __)
( (_ \) \/ (  )( (  O )\ /\ / )(  )   / ) _) 
 \___/\____/ (__) \__/ (_/\_)(__)(__\_)(____)
`)

// copied from cobra:.
const defaultVersionTemplate = `{{with .DisplayName}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}

`

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if term.IsTerminal(os.Stdout.Fd()) {
		var b bytes.Buffer
		w := colorprofile.NewWriter(os.Stdout, os.Environ())
		w.Forward = &b
		_, _ = w.WriteString(versionBit.String())
		rootCmd.SetVersionTemplate(b.String() + "\n" + defaultVersionTemplate)
	}
	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithVersion(version.Version),
		fang.WithNotifySignal(os.Interrupt),
	); err != nil {
		os.Exit(1)
	}
}

// handleInitConfig function    处理初始化配置文件.
func handleInitConfig() error {
	configPath := ".gutowire.yaml"
	if configFile != "" {
		configPath = configFile
	}

	if err := config.GenerateExampleConfig(configPath); err != nil {
		return fmt.Errorf("生成配置文件失败: %w", err)
	}

	fmt.Printf("✓ 配置文件已生成: %s\n", configPath)
	fmt.Println("\n你可以编辑此文件来自定义配置")
	return nil
}

// handleWatch function    处理 watch 模式.
func handleWatch(wirePath, searchPath string, opts []config.Option) error {
	fmt.Println("🔍 启动 Watch 模式...")

	// 首先执行一次生成
	if err := runner.RunAutoWire(wirePath, opts...); err != nil {
		return fmt.Errorf("初始生成失败: %w", err)
	}

	fmt.Println("✓ 初始生成完成")

	// 创建 watcher
	w, err := watcher.New(wirePath, []string{"*.gen.go", "wire_gen.go"}, opts...)
	if err != nil {
		return fmt.Errorf("创建监听器失败: %w", err)
	}
	defer w.Close()

	// 开始监听
	if searchPath == "" {
		searchPath = "."
	}
	return w.Watch(searchPath)
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gutowire.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.PersistentFlags().StringVarP(&wirePath, "wire_path", "w", "", "Wire 配置文件生成路径")
	rootCmd.PersistentFlags().StringVarP(&scope, "scope", "s", "", "依赖搜索范围(目录路径),不填则全局搜索")
	rootCmd.PersistentFlags().StringVarP(&pkg, "pkg", "p", "", "生成文件的包名")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "配置文件路径 (默认: .gutowire.yaml)")
	rootCmd.PersistentFlags().BoolVar(&watch, "watch", false, "启用 watch 模式，自动监听文件变化")
	rootCmd.PersistentFlags().BoolVar(&noCache, "no-cache", false, "禁用缓存")
	rootCmd.PersistentFlags().BoolVar(&initConfig, "init", false, "生成示例配置文件")
}
