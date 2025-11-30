// Package version 管理 gutowire 的版本信息。
// 版本号可以在编译时通过 -ldflags 设置，或从 Go 模块信息中读取。
package version

import "runtime/debug"

// Build-time parameters set via -ldflags

// Version represents the current version of the application.
// It is set at build time via -ldflags, or defaults to "devel".
var Version = "devel"

// init function    初始化版本信息
// A user may install crush using `go install github.com/charmbracelet/crush@latest`.
// without -ldflags, in which case the version above is unset. As a workaround
// we use the embedded build version that *is* set when using `go install` (and
// is only set for `go install` and not for `go build`).
func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	mainVersion := info.Main.Version
	if mainVersion != "" && mainVersion != "(devel)" {
		Version = mainVersion
	}
}
