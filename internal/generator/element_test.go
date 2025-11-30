package generator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestGetImplement(t *testing.T) {
	src := `package test

type Writer interface {
	Write([]byte) (int, error)
}

type MyWriter struct{}

var _ Writer = &MyWriter{}
var _ Writer = (*MyWriter)(nil)
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("解析代码失败: %v", err)
	}

	result := getImplement(f)

	if len(result) == 0 {
		t.Error("getImplement() 应该找到接口实现")
	}

	if impl, ok := result["MyWriter"]; !ok || impl != "Writer" {
		t.Errorf("getImplement() = %v, want MyWriter -> Writer", result)
	}
}

func TestGetImplement_NoImplementation(t *testing.T) {
	src := `package test

type Writer interface {
	Write([]byte) (int, error)
}

type MyWriter struct{}
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("解析代码失败: %v", err)
	}

	result := getImplement(f)

	if len(result) != 0 {
		t.Errorf("getImplement() = %v, want empty map", result)
	}
}

func TestParseValueSpec(t *testing.T) {
	tests := []struct {
		name          string
		src           string
		wantImplType  string
		wantInterface string
	}{
		{
			name:          "指针类型实现",
			src:           "var _ Writer = &MyWriter{}",
			wantImplType:  "MyWriter",
			wantInterface: "Writer",
		},
		{
			name:          "值类型实现",
			src:           "var _ Writer = MyWriter{}",
			wantImplType:  "MyWriter",
			wantInterface: "Writer",
		},
		{
			name:          "非接口实现",
			src:           "var x = 10",
			wantImplType:  "",
			wantInterface: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := "package test\n\ntype Writer interface{}\ntype MyWriter struct{}\n\n" + tt.src
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
			if err != nil {
				t.Fatalf("解析代码失败: %v", err)
			}

			// 查找最后一个声明（我们添加的测试声明）
			var implType, interfaceName string
			for i := len(f.Decls) - 1; i >= 0; i-- {
				if gd, ok := f.Decls[i].(*ast.GenDecl); ok {
					for _, spec := range gd.Specs {
						implType, interfaceName = parseValueSpec(spec)
						goto done
					}
				}
			}
		done:
			if implType != tt.wantImplType || interfaceName != tt.wantInterface {
				t.Errorf("parseValueSpec() = (%q, %q), want (%q, %q)",
					implType, interfaceName, tt.wantImplType, tt.wantInterface)
			}
		})
	}
}

func TestExtractImplTypeName(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "复合字面量",
			src:  "MyWriter{}",
			want: "MyWriter",
		},
		{
			name: "指针复合字面量",
			src:  "&MyWriter{}",
			want: "MyWriter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := "package test\n\ntype MyWriter struct{}\n\nvar x = " + tt.src
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
			if err != nil {
				t.Fatalf("解析代码失败: %v", err)
			}

			for _, decl := range f.Decls {
				if gd, ok := decl.(*ast.GenDecl); ok {
					for _, spec := range gd.Specs {
						if vs, ok := spec.(*ast.ValueSpec); ok && len(vs.Values) > 0 {
							got := extractImplTypeName(vs.Values[0])
							if got != tt.want {
								t.Errorf("extractImplTypeName() = %q, want %q", got, tt.want)
							}
							return
						}
					}
				}
			}
		})
	}
}
