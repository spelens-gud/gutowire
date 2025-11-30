package internal

import (
	"go/ast"
	"go/token"
)

// tmpDecl 临时声明信息，用于解析 AST 时存储类型或函数的信息.
type tmpDecl struct {
	docs     string        // 文档注释（包含 @autowire 注解）
	name     string        // 名称
	isFunc   bool          // 是否为函数
	typeSpec *ast.TypeSpec // 类型规范（如果是类型声明）
}

// getImplement 分析文件中的接口实现声明
// 查找类似 var _ io.Writer = &myWriter{} 的接口实现声明
// 返回 map[实现类型名]接口名
//
// 这种声明方式是 Go 中常用的编译时接口实现检查.
func getImplement(f *ast.File) map[string]string {
	ret := make(map[string]string)

	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, sp := range gd.Specs {
			vs, ok := sp.(*ast.ValueSpec)
			// 检查是否为 var _ InterfaceName = ... 格式
			if !ok || vs.Names[0].Name != "_" || vs.Type == nil || len(vs.Values) != 1 {
				continue
			}

			var id *ast.Ident

			// 解析赋值表达式，提取实现类型
			switch t := vs.Values[0].(type) {
			case *ast.CompositeLit:
				// 情况1: var _ Interface = Type{}
				id, ok = t.Type.(*ast.Ident)
				if !ok {
					continue
				}
			case *ast.UnaryExpr:
				// 情况2: var _ Interface = &Type{}
				if t.Op != token.AND {
					continue
				}
				cl, ok := t.X.(*ast.CompositeLit)
				if !ok {
					continue
				}
				id, ok = cl.Type.(*ast.Ident)
				if !ok {
					continue
				}
			default:
				continue
			}

			// 提取接口名
			if imp, ok := vs.Type.(*ast.Ident); !ok {
				continue
			} else {
				ret[id.Name] = imp.Name
			}
		}
	}
	return ret
}
