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
	"github.com/spelens-gud/gutowire/internal"
	"github.com/spelens-gud/gutowire/internal/version"
	"github.com/spf13/cobra"
)

const (
	commandName = "gutowire"
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
		// 构建配置选项
		var opts []internal.Option

		// 应用包名配置
		if pkg != "" {
			opts = append(opts, internal.WithPkg(pkg))
		}

		// 应用搜索路径配置
		if scope != "" {
			opts = append(opts, internal.WithSearchPath(scope))
		}

		// 从位置参数或标志获取生成路径
		if wirePath == "" && len(args) > 0 {
			wirePath = args[0]
		}

		// 验证必需参数
		if wirePath == "" {
			return fmt.Errorf("必须指定 Wire 配置文件生成路径\n使用方式: %s [flags] <生成路径>", commandName)
		}

		// 添加默认的初始化配置
		opts = append(opts, internal.InitStruct())

		// 执行自动装配
		if err := internal.RunAutoWire(wirePath, opts...); err != nil {
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
}
