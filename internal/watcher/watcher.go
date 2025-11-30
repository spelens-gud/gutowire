// Package watcher 提供文件监听功能，支持自动重新生成代码
package watcher

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spelens-gud/gutowire/internal/config"
	"github.com/spelens-gud/gutowire/internal/runner"
)

// Watcher struct    文件监听器.
type Watcher struct {
	watcher        *fsnotify.Watcher
	genPath        string
	opts           []config.Option
	ignorePatterns []string
	debounceTime   time.Duration
	lastRun        time.Time
}

// New function    创建新的文件监听器.
func New(genPath string, ignorePatterns []string, opts ...config.Option) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("创建文件监听器失败: %w", err)
	}

	return &Watcher{
		watcher:        w,
		genPath:        genPath,
		opts:           opts,
		ignorePatterns: ignorePatterns,
		debounceTime:   500 * time.Millisecond, // 防抖时间
		lastRun:        time.Now(),
	}, nil
}

// Watch method    开始监听.
func (w *Watcher) Watch(searchPath string) error {
	log.Printf("> 开始监听目录: %s <", searchPath)
	log.Printf("! 提示: 修改 .go 文件后将自动重新生成代码")
	log.Printf("⏸  按 Ctrl+Z 停止监听\n")

	// 递归添加目录到监听列表
	if err := w.addRecursive(searchPath); err != nil {
		return fmt.Errorf("添加监听目录失败: %w", err)
	}

	// 处理事件
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("x 监听错误: %v", err)
		}
	}
}

// handleEvent method    处理文件变更事件.
func (w *Watcher) handleEvent(event fsnotify.Event) {
	// 忽略非 Go 文件
	if !strings.HasSuffix(event.Name, ".go") {
		return
	}

	// 忽略生成的文件
	if w.shouldIgnore(event.Name) {
		return
	}

	// 只处理写入和创建事件
	if event.Op&fsnotify.Write != fsnotify.Write && event.Op&fsnotify.Create != fsnotify.Create {
		return
	}

	// 防抖：避免短时间内多次触发
	now := time.Now()
	if now.Sub(w.lastRun) < w.debounceTime {
		return
	}
	w.lastRun = now

	log.Printf("\n> 检测到文件变更: %s", event.Name)
	log.Printf(">>>>>>> 正在重新生成代码 >>>>>>\n")

	// 执行代码生成
	if err := runner.RunAutoWire(w.genPath, w.opts...); err != nil {
		log.Printf("x 生成失败: %v\n", err)
	} else {
		log.Printf("✓ 生成成功\n")
	}
}

// shouldIgnore method    检查是否应该忽略该文件.
func (w *Watcher) shouldIgnore(path string) bool {
	base := filepath.Base(path)

	// 忽略生成的文件
	if strings.HasPrefix(base, "autowire_") || base == "wire_gen.go" {
		return true
	}

	// 忽略测试文件
	if strings.HasSuffix(base, "_test.go") {
		return true
	}

	// 检查自定义忽略模式
	for _, pattern := range w.ignorePatterns {
		matched, _ := filepath.Match(pattern, base)
		if matched {
			return true
		}
	}

	return false
}

// addRecursive method    递归添加目录到监听列表.
func (w *Watcher) addRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过非目录
		if !info.IsDir() {
			return nil
		}

		// 跳过隐藏目录和特殊目录
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") || base == "vendor" || base == "testdata" {
			return filepath.SkipDir
		}

		// 添加到监听列表
		if err := w.watcher.Add(path); err != nil {
			return fmt.Errorf("添加监听目录 %s 失败: %w", path, err)
		}

		return nil
	})
}

// Close method    关闭监听器.
func (w *Watcher) Close() error {
	return w.watcher.Close()
}
