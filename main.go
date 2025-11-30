// Package main 是 gutowire 工具的入口点。
// gutowire 是一个基于 Google Wire 的依赖注入代码生成工具，
// 通过扫描 @autowire 注解自动生成 Wire 配置文件。
package main

/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/

import "github.com/spelens-gud/gutowire/cmd"

func main() {
	cmd.Execute()
}
