package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckFileType(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"普通 Go 文件", "main.go", true},
		{"测试文件应跳过", "main_test.go", false},
		{"非 Go 文件", "main.txt", false},
		{"没有扩展名", "main", false},
		{"隐藏的 Go 文件", ".hidden.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckFileType(tt.filename); got != tt.want {
				t.Errorf("CheckFileType(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestAppendPkg(t *testing.T) {
	tests := []struct {
		name string
		pkg  string
		sel  string
		want string
	}{
		{"包名和选择器", "fmt", "Println", "fmt.Println"},
		{"空包名", "", "Println", "Println"},
		{"空选择器", "fmt", "", "fmt."},
		{"都为空", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AppendPkg(tt.pkg, tt.sel); got != tt.want {
				t.Errorf("AppendPkg(%q, %q) = %q, want %q", tt.pkg, tt.sel, got, tt.want)
			}
		})
	}
}

func TestGetPathGoPkgName(t *testing.T) {
	// 创建临时目录和测试文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := []byte("package testpkg\n\nfunc main() {}\n")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	got, err := GetPathGoPkgName(tmpDir)
	if err != nil {
		t.Fatalf("GetPathGoPkgName() error = %v", err)
	}

	want := "testpkg"
	if got != want {
		t.Errorf("GetPathGoPkgName() = %q, want %q", got, want)
	}
}

func TestGetPathGoPkgName_EmptyDir(t *testing.T) {
	// 空目录应该返回目录名作为包名
	tmpDir := t.TempDir()

	got, err := GetPathGoPkgName(tmpDir)
	if err != nil {
		t.Fatalf("GetPathGoPkgName() error = %v", err)
	}

	// 应该返回目录的基础名称
	want := filepath.Base(tmpDir)
	if got != want {
		t.Errorf("GetPathGoPkgName() = %q, want %q", got, want)
	}
}

func TestGetPathGoPkgName_NonExistentDir(t *testing.T) {
	_, err := GetPathGoPkgName("/nonexistent/directory")
	if err == nil {
		t.Error("GetPathGoPkgName() 应该返回错误，但没有")
	}
}
