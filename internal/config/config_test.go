package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWithPkg(t *testing.T) {
	opt := &Opt{}
	WithPkg("testpkg")(opt)

	if opt.Pkg != "testpkg" {
		t.Errorf("WithPkg() 设置的包名 = %q, want %q", opt.Pkg, "testpkg")
	}
}

func TestInitStruct(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "无参数",
			input: []string{},
			want:  []string{"*"},
		},
		{
			name:  "单个类型",
			input: []string{"Zoo"},
			want:  []string{"Zoo"},
		},
		{
			name:  "多个类型",
			input: []string{"Zoo", "App"},
			want:  []string{"Zoo", "App"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := &Opt{}
			InitStruct(tt.input...)(opt)

			if len(opt.InitWire) != len(tt.want) {
				t.Errorf("InitStruct() 长度 = %d, want %d", len(opt.InitWire), len(tt.want))
				return
			}

			for i, v := range opt.InitWire {
				if v != tt.want[i] {
					t.Errorf("InitStruct()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestWithSearchPath(t *testing.T) {
	opt := &Opt{}
	WithSearchPath("/test/path")(opt)

	if opt.SearchPath != "/test/path" {
		t.Errorf("WithSearchPath() 设置的路径 = %q, want %q", opt.SearchPath, "/test/path")
	}
}

func TestNewGenOpt(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 创建一个测试 Go 文件
	testFile := filepath.Join(tmpDir, "test.go")
	content := []byte("package testpkg\n")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	opt := NewGenOpt(tmpDir)

	if opt.GenPath != tmpDir {
		t.Errorf("GenPath = %q, want %q", opt.GenPath, tmpDir)
	}

	if opt.Pkg != "testpkg" {
		t.Errorf("Pkg = %q, want %q", opt.Pkg, "testpkg")
	}
}

func TestNewGenOpt_WithOptions(t *testing.T) {
	tmpDir := t.TempDir()

	opt := NewGenOpt(
		tmpDir,
		WithPkg("custompkg"),
		WithSearchPath("/custom/path"),
		InitStruct("Zoo", "App"),
	)

	if opt.Pkg != "custompkg" {
		t.Errorf("Pkg = %q, want %q", opt.Pkg, "custompkg")
	}

	if opt.SearchPath != "/custom/path" {
		t.Errorf("SearchPath = %q, want %q", opt.SearchPath, "/custom/path")
	}

	if len(opt.InitWire) != 2 {
		t.Errorf("InitWire 长度 = %d, want 2", len(opt.InitWire))
	}
}
