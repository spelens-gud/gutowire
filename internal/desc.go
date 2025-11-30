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
// 返回 map[实现类型名]接口名.
func getImplement(f *ast.File) map[string]string {
	ret := make(map[string]string)

	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, sp := range gd.Specs {
			if implType, interfaceName := parseValueSpec(sp); implType != "" && interfaceName != "" {
				ret[implType] = interfaceName
			}
		}
	}
	return ret
}

// parseValueSpec 解析变量声明规范，提取接口实现信息.
func parseValueSpec(spec ast.Spec) (implType, interfaceName string) {
	vs, ok := spec.(*ast.ValueSpec)
	// 检查是否为 var _ InterfaceName = ... 格式
	if !ok || len(vs.Names) == 0 || vs.Names[0].Name != "_" || vs.Type == nil || len(vs.Values) != 1 {
		return "", ""
	}

	// 提取接口名称
	interfaceIdent, ok := vs.Type.(*ast.Ident)
	if !ok {
		return "", ""
	}

	// 解析实现类型名称
	implTypeName := extractImplTypeName(vs.Values[0])

	return implTypeName, interfaceIdent.Name
}

// extractImplTypeName 从表达式中提取实现类型名称.
func extractImplTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.CompositeLit:
		// 情况1: var _ Interface = Type{}
		if id, ok := t.Type.(*ast.Ident); ok {
			return id.Name
		}
	case *ast.UnaryExpr:
		// 情况2: var _ Interface = &Type{}
		return extractFromUnaryExpr(t)
	}
	return ""
}

// extractFromUnaryExpr 从一元表达式中提取类型名称.
func extractFromUnaryExpr(ue *ast.UnaryExpr) string {
	if ue.Op != token.AND {
		return ""
	}

	cl, ok := ue.X.(*ast.CompositeLit)
	if !ok {
		return ""
	}

	if id, ok := cl.Type.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}
