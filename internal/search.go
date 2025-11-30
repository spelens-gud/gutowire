package internal

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/stoewer/go-strcase"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// autoWireSearcher 自动装配搜索器，负责扫描和收集所有需要注入的组件.
type autoWireSearcher struct {
	sets           []string                      // 所有 Set 的名称列表
	genPath        string                        // 生成文件的路径
	pkg            string                        // 包名
	elementMap     map[string]map[string]element // Set名称 -> (组件路径 -> 组件信息)
	modBase        string                        // Go module 的基础路径
	initElements   []element                     // 标记为 init 的元素列表
	configElements []element                     // 标记为 config 的元素列表
	initWire       []string                      // 需要初始化的类型
	wg             errgroup.Group                // 并发控制
	mu             sync.Mutex                    // 并发安全锁
}

// newAutoWireSearcher function    创建一个自动装配搜索器.
func newAutoWireSearcher(genPath string, modBase string, initWire []string, pkg string) *autoWireSearcher {
	return &autoWireSearcher{
		genPath:    genPath,
		modBase:    modBase,
		initWire:   initWire,
		elementMap: make(map[string]map[string]element),
		pkg:        pkg,
	}
}

// SearchAllPath 递归扫描指定目录下的所有 Go 文件
// 跳过 vendor 和 testdata 目录，跳过测试文件.
func (sc *autoWireSearcher) SearchAllPath(file string) (err error) {
	return filepath.Walk(file, func(path string, f os.FileInfo, _ error) error {
		fn := f.Name()

		// 跳过 vendor 和 testdata 目录
		if f.IsDir() && (fn == "vendor" || fn == "testdata") {
			return filepath.SkipDir
		}

		// 只处理 .go 文件，跳过测试文件
		if f.IsDir() || !checkFileType(fn) {
			return nil
		}

		// 并发处理每个文件
		sc.wg.Go(func() error {
			return sc.searchWire(path)
		})

		// 等待当前文件处理完成再继续
		return sc.wg.Wait()
	})
}

// searchWire 扫描单个 Go 文件，查找并解析 @autowire 注解.
func (sc *autoWireSearcher) searchWire(file string) error {
	// 读取文件内容
	//nolint:gosec
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("读取文件 %s 失败: %w", file, err)
	}

	// 快速检查：如果文件中没有 @autowire 标记，直接跳过
	if !bytes.Contains(data, []byte(wireTag)) {
		return nil
	}

	// 解析 Go 源文件的 AST
	parseFile, err := parser.ParseFile(token.NewFileSet(), "", data, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("解析文件 %s 失败: %w", file, err)
	}

	// 检查是否会导致循环导入
	if sc.wouldCauseCircularImport(parseFile, file) {
		return nil
	}

	// 收集所有带 @autowire 注解的声明
	matchDecls := sc.collectAnnotatedDecls(parseFile)

	// 获取接口实现关系
	implementMap := getImplement(parseFile)

	// 解析每个声明的注解
	sc.parseAnnotations(matchDecls, file, parseFile, implementMap)

	return nil
}

// wouldCauseCircularImport 检查是否会引发循环导入.
func (sc *autoWireSearcher) wouldCauseCircularImport(parseFile *ast.File, file string) bool {
	genPkgPath := fmt.Sprintf(`"%s"`, sc.getPkgPath(filepath.Join(sc.genPath, "...")))
	for _, imp := range parseFile.Imports {
		if imp.Path.Value == genPkgPath {
			log.Printf("[warn] 包 %s (来自 %s) 已导入生成目标包，跳过以避免循环依赖", parseFile.Name.Name, file)
			return true
		}
	}
	return false
}

// collectAnnotatedDecls 收集所有带 @autowire 注解的声明.
func (sc *autoWireSearcher) collectAnnotatedDecls(parseFile *ast.File) []tmpDecl {
	var matchDecls []tmpDecl

	for _, decl := range parseFile.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			// 只处理 type 壀查
			if d.Tok.String() != "type" {
				continue
			}
			matchDecls = append(matchDecls, sc.collectTypeDecls(d)...)

		case *ast.FuncDecl:
			// 处理函数声明(构造函数)
			if strings.Contains(d.Doc.Text(), wireTag) {
				matchDecls = append(matchDecls, tmpDecl{
					docs:   d.Doc.Text(),
					name:   d.Name.Name,
					isFunc: true,
				})
			}
		}
	}

	return matchDecls
}

// collectTypeDecls 收集类型声明中的注解.
func (sc *autoWireSearcher) collectTypeDecls(d *ast.GenDecl) []tmpDecl {
	var result []tmpDecl

	// 情况1: 单个类型声明
	// @autowire()
	// type Some struct{}
	if len(d.Specs) == 1 && strings.Contains(d.Doc.Text(), wireTag) {
		if id, ok := d.Specs[0].(*ast.TypeSpec); ok {
			result = append(result, tmpDecl{
				docs:     d.Doc.Text(),
				name:     id.Name.Name,
				isFunc:   false,
				typeSpec: id,
			})
		}
		return result
	}

	// 情况2: 类型组声明
	// type (
	//     @autowire()
	//     A struct{}
	//     @autowire()
	//     B struct{}
	// )
	for _, sp := range d.Specs {
		if id, ok := sp.(*ast.TypeSpec); ok && strings.Contains(id.Doc.Text(), wireTag) {
			result = append(result, tmpDecl{
				docs:     id.Doc.Text(),
				name:     id.Name.Name,
				isFunc:   false,
				typeSpec: id,
			})
		}
	}

	return result
}

// parseAnnotations 解析声明的注解.
func (sc *autoWireSearcher) parseAnnotations(matchDecls []tmpDecl, file string,
	parseFile *ast.File, implementMap map[string]string) {
	for _, decl := range matchDecls {
		lines := strings.Split(decl.docs, "\n")
		for _, c := range lines {
			sc.analysisWireTag(strings.TrimSpace(c), file, &decl,
				parseFile, implementMap)
		}
	}
}

// getPkgPath 获取文件的完整包导入路径
// 这是 getPkgPath 的包装方法，使用搜索器的 modBase.
func (sc *autoWireSearcher) getPkgPath(filePath string) (pkgPath string) {
	return getPkgPath(filePath, sc.modBase)
}

// analysisWireTag 解析单行 @autowire 注解
// 这是注解解析的核心函数，支持多种注解格式：
// - @autowire(set=animals) - 基础用法
// - @autowire.init(set=zoo) - 生成初始化函数
// - @autowire.config(set=config) - 配置注入
// - @autowire(set=animals,FlyAnimal) - 接口绑定
// - @autowire(set=animals,new=CustomConstructor) - 自定义构造函数.
func (sc *autoWireSearcher) analysisWireTag(tag, filePath string, decl *tmpDecl, f *ast.File,
	implementMap map[string]string) {
	// 检查是否为 @autowire 注解
	if !strings.HasPrefix(tag, wireTag) {
		return
	}

	itemFunc, tagStr := sc.parseTagSuffix(tag)

	// 检查括号格式
	if !strings.HasPrefix(tagStr, "(") || !strings.HasSuffix(tagStr, ")") {
		return
	}

	// 解析注解参数
	options := sc.parseTagOptions(tagStr)

	// 创建组件元素
	wireElement := sc.createWireElement(decl, f, filePath)

	// 确定构造函数
	sc.determineConstructor(&wireElement, decl, f)

	// 确定 Set 名称
	setName := sc.determineSetName(options)

	// 解析其他选项
	itemFunc = sc.parseOptions(options, &wireElement, f, itemFunc)

	// 处理特殊函数标记
	setName = sc.handleSpecialFunctions(itemFunc, setName, &wireElement, decl)

	// 添加接口实现关系
	sc.addInterfaceImplementations(&wireElement, implementMap, decl.name)

	// 延迟执行：将组件添加到 elementMap
	sc.addElementToMap(setName, filePath, wireElement, decl.name)
}

// parseTagSuffix 解析 .init 或 .config 后缀.
func (sc *autoWireSearcher) parseTagSuffix(tag string) (itemFunc, tagStr string) {
	tagStr = tag[len(wireTag):] // 去掉 @autowire 前缀

	// 解析 .init 或 .config 后缀
	// 例如: @autowire.init(set=zoo)
	if len(tagStr) > 0 && tagStr[0] == '.' {
		idx := strings.IndexRune(tagStr, '(')
		if idx != -1 {
			itemFunc = tagStr[1:idx] // 提取 init 或 config
			tagStr = tagStr[idx:]
		}
	}
	return
}

// parseTagOptions 解析注解参数.
func (sc *autoWireSearcher) parseTagOptions(tagStr string) map[string]string {
	options := make(map[string]string)
	content := strings.TrimPrefix(strings.TrimSuffix(tagStr, ")"), "(")

	for _, s := range strings.Split(content, ",") {
		if s = strings.TrimSpace(s); len(s) == 0 {
			continue
		}
		spo := strings.Split(s, "=")
		v := ""
		if len(spo) > 1 {
			v = strings.TrimSpace(spo[1])
		}
		options[strings.TrimSpace(spo[0])] = v
	}
	return options
}

// createWireElement 创建组件元素.
func (sc *autoWireSearcher) createWireElement(decl *tmpDecl, f *ast.File, filePath string) element {
	return element{
		name:    decl.name,
		pkg:     f.Name.Name,
		pkgPath: sc.getPkgPath(filePath),
	}
}

// determineConstructor 确定构造函数.
func (sc *autoWireSearcher) determineConstructor(wireElement *element, decl *tmpDecl, f *ast.File) {
	if decl.isFunc {
		// 如果是函数声明，函数本身就是构造函数
		wireElement.constructor = decl.name
	} else {
		// 如果是结构体，查找 New<Name> 或 Init<Name> 构造函数
		for _, constructorPrefix := range []string{"Init", "New"} {
			if ct, ok := f.Scope.Objects[constructorPrefix+decl.name]; ok && ct.Kind == ast.Fun {
				wireElement.constructor = constructorPrefix + decl.name
				break
			}
		}
	}
}

// determineSetName 确定 Set 名称.
func (sc *autoWireSearcher) determineSetName(options map[string]string) string {
	if len(options["set"]) == 0 {
		return "unknown"
	}
	return strcase.LowerCamelCase(options["set"])
}

// parseOptions 解析其他选项.
func (sc *autoWireSearcher) parseOptions(options map[string]string, wireElement *element, f *ast.File,
	itemFunc string) string {
	resultFunc := itemFunc

	for key, value := range options {
		switch key {
		case "init", "config":
			// 如果在参数中指定 init 或 config
			resultFunc = key
		case "set":
			// set 已经处理过，跳过
			continue
		case "new":
			// 自定义构造函数名称
			if ct, ok := f.Scope.Objects[value]; ok && ct.Kind == ast.Fun {
				wireElement.constructor = value
			}
			continue
		default:
			// 其他参数视为接口名称
			wireElement.implements = append(wireElement.implements, key)
		}
	}
	return resultFunc
}

// handleSpecialFunctions 处理特殊函数标记.
func (sc *autoWireSearcher) handleSpecialFunctions(itemFunc, setName string, wireElement *element,
	decl *tmpDecl) string {
	resultSetName := setName

	switch itemFunc {
	case "init":
		// @autowire.init - 标记为初始化入口
		wireElement.initWire = true
		resultSetName = "init"
	case "config":
		// @autowire.config - 配置注入模式
		sc.handleConfigFunction(wireElement, decl)
		resultSetName = "config"
	}
	return resultSetName
}

// handleConfigFunction 处理 config 特殊函数标记.
func (sc *autoWireSearcher) handleConfigFunction(wireElement *element, decl *tmpDecl) {
	if !sc.isValidConfigDeclaration(decl) {
		return
	}

	wireElement.configWire = true

	// 提取所有导出字段（首字母大写）
	st := decl.typeSpec.Type.(*ast.StructType)
	for _, f := range st.Fields.List {
		fieldName := sc.extractFieldName(f)
		// 只收集导出字段
		if sc.isExportedField(fieldName) {
			wireElement.fields = append(wireElement.fields, fieldName)
		}
	}
}

// isValidConfigDeclaration 检查是否为有效的配置声明.
func (sc *autoWireSearcher) isValidConfigDeclaration(decl *tmpDecl) bool {
	if decl.typeSpec == nil {
		return false
	}

	st, isStruct := decl.typeSpec.Type.(*ast.StructType)
	return isStruct && st.Fields != nil && len(st.Fields.List) > 0
}

// extractFieldName 提取字段名称.
func (sc *autoWireSearcher) extractFieldName(f *ast.Field) string {
	fieldName := fmt.Sprintf("%s", f.Type)
	if f.Names != nil {
		fieldName = f.Names[0].String()
	}
	return fieldName
}

// isExportedField 检查字段是否为导出字段（首字母大写.
func (sc *autoWireSearcher) isExportedField(fieldName string) bool {
	return len(fieldName) > 0 && fieldName[0] >= 'A' && fieldName[0] <= 'Z'
}

// addInterfaceImplementations 添加接口实现关系.
func (sc *autoWireSearcher) addInterfaceImplementations(wireElement *element,
	implementMap map[string]string, name string) {
	if impl := implementMap[name]; impl != "" && !slices.Contains(wireElement.implements, impl) {
		wireElement.implements = append(wireElement.implements, impl)
	}
}

// addElementToMap 将组件添加到 elementMap.
func (sc *autoWireSearcher) addElementToMap(setName, filePath string, wireElement element, name string) {
	pkgPath := sc.getPkgPath(filePath)

	log.Printf("收集到 wire 对象 [ %sSet ] : %s\n", strcase.LowerCamelCase(setName), wireElement.pkg+"."+wireElement.name)
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.elementMap[setName] == nil {
		sc.elementMap[setName] = make(map[string]element)
	}
	sc.elementMap[setName][path.Join(pkgPath, name)] = wireElement
}

// write 执行代码生成的主流程
// 生成所有 Wire 配置文件：
// 1. 为每个 Set 生成独立的文件（autowire_animals.go, autowire_zoo.go 等）
// 2. 生成汇总文件（autowire_sets.go）
// 3. 生成初始化入口文件(wire.gen.go).
func (sc *autoWireSearcher) write() error {
	log.Printf("正在生成文件到目录 [ %s ] ...", sc.genPath)
	sc.sets = nil

	// 确保目标目录存在
	if err := os.MkdirAll(sc.genPath, 0750); err != nil {
		return fmt.Errorf("创建目录 %s 失败: %w", sc.genPath, err)
	}

	// 清理旧文件
	if err := sc.clean(); err != nil {
		return fmt.Errorf("清理旧文件失败: %w", err)
	}

	// 并发生成每个 Set 的文件
	for set, m := range sc.elementMap {
		sc.wg.Go(func() error {
			return sc.writeSet(set, m)
		})
	}

	// 等待所有 Set 文件生成完成
	if err := sc.wg.Wait(); err != nil {
		return fmt.Errorf("生成 Set 文件失败: %w", err)
	}

	// 生成汇总文件和初始化文件
	return sc.writeSets()
}

// clean 清理之前生成的文件
// 删除所有 autowire_*.go 和 wire_gen.go 文件，为新的生成做准备.
func (sc *autoWireSearcher) clean() error {
	entries, err := os.ReadDir(sc.genPath)
	if err != nil {
		return fmt.Errorf("读取目录 %s 失败: %w", sc.genPath, err)
	}
	if len(entries) == 0 {
		return nil
	}

	// 删除 wire_gen.go（由 wire 命令生成的文件）
	_ = os.Remove(filepath.Join(sc.genPath, "wire_gen.go"))

	// 删除所有 autowire_*.go 文件
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, filePrefix+"_") && strings.HasSuffix(name, ".go") {
			_ = os.Remove(filepath.Join(sc.genPath, name))
		}
	}
	return nil
}

// writeSet 为单个 Set 生成配置文件
// 例如：为 animals Set 生成 autowire_animals.go
//
// set: Set 的名称（如 "animals"）
// elements: 该 Set 包含的所有组件
func (sc *autoWireSearcher) writeSet(set string, elements map[string]element) error {
	pkgMap := make(map[string]map[string]string) // 用于处理包名冲突

	setName := cases.Title(language.Und, cases.NoLower).String(strcase.UpperCamelCase(set)) + "Set"
	fileName := filepath.Join(sc.genPath, filePrefix+"_"+strcase.SnakeCase(set)+".go")

	log.Printf("正在生成 %s [ %s ]", setName, fileName)

	// 收集所有元素的 key 并排序，保证生成顺序稳定
	order := SortedKeys(elements)

	// 处理包名冲突
	sc.resolvePackageConflicts(elements, pkgMap, order)

	// 生成 Wire 配置代码
	data, importPkg := sc.generateWireConfig(setName, elements, order)

	// 写入文件
	if err := sc.writeConfigFile(fileName, data, importPkg); err != nil {
		return err
	}

	// 记录 Set 名称
	sc.mu.Lock()
	sc.sets = append(sc.sets, setName)
	sc.mu.Unlock()

	return nil
}

// resolvePackageConflicts 处理包名冲突.
func (sc *autoWireSearcher) resolvePackageConflicts(elements map[string]element, pkgMap map[string]map[string]string,
	order []string) {
	for _, elementKey := range order {
		elem := elements[elementKey]
		pkg, ok := pkgMap[elem.pkg][elem.pkgPath]

		// 第一次遇到这个包名
		if len(pkgMap[elem.pkg]) == 0 {
			pkg = elem.pkg
			pkgMap[elem.pkg] = map[string]string{
				elem.pkgPath: elem.pkg,
			}
			ok = true
		}

		if ok {
			elem.pkg = pkg
			elements[elementKey] = elem
			continue
		}

		// 包名冲突，添加数字后缀
		fixPkgDuplicate := len(pkgMap[elem.pkg]) + 1
		newPkg := elem.pkg + strconv.Itoa(fixPkgDuplicate)
		pkgMap[elem.pkg][elem.pkgPath] = newPkg
		elem.pkg = newPkg
		elements[elementKey] = elem
	}
}

// generateWireConfig 生成 Wire 配置代码.
func (sc *autoWireSearcher) generateWireConfig(setName string, elements map[string]element,
	order []string) (wireSet, []*ast.ImportSpec) {
	var importPkg []*ast.ImportSpec
	pathPkg := sc.getPkgPath(filepath.Join(sc.genPath, filePrefix+"_"+
		strcase.SnakeCase(strings.TrimSuffix(setName, "Set"))+".go"))

	data := wireSet{
		Package: sc.pkg,
		SetName: setName,
	}

	// 为每个元素生成 Wire 配置代码
	for _, key := range order {
		var wireItem []string
		elem := elements[key]

		// 如果元素在同一个包中，不需要包前缀
		if elem.pkgPath == pathPkg {
			elem.pkg = ""
		}

		stName := appendPkg(elem.pkg, elem.name)

		if elem.configWire {
			// 配置模式：使用 wire.FieldsOf 提取字段
			sc.handleConfigWireElement(&elem, &wireItem, stName)
		} else {
			// 普通模式
			sc.handleNormalWireElement(&elem, &wireItem, stName)
		}

		data.Items = append(data.Items, strings.Join(wireItem, ",\n\t"))

		// 如果需要导入包，添加到 import 列表
		if len(elem.pkg) > 0 {
			imp := sc.createImportSpec(&elem)
			importPkg = append(importPkg, imp)
		}
	}

	return data, importPkg
}

// handleConfigWireElement 处理配置类型的 Wire 元素.
func (sc *autoWireSearcher) handleConfigWireElement(elem *element, wireItem *[]string, stName string) {
	slices.Sort(elem.fields)
	// 构建字段列表字符串
	fieldsList := Map(elem.fields, func(field string) string {
		return fmt.Sprintf(`"%s"`, field)
	})
	fieldsStr := strings.Join(fieldsList, ", ")
	*wireItem = append(*wireItem, fmt.Sprintf(`wire.FieldsOf(new(*%s), %s)`, stName, fieldsStr))
	sc.mu.Lock()
	sc.configElements = append(sc.configElements, *elem)
	sc.mu.Unlock()
}

// handleNormalWireElement 处理普通类型的 Wire 元素.
func (sc *autoWireSearcher) handleNormalWireElement(elem *element, wireItem *[]string, stName string) {
	if elem.constructor != "" {
		// 有构造函数，直接使用构造函数
		*wireItem = append(*wireItem, appendPkg(elem.pkg, elem.constructor))
	} else {
		// 没有构造函数，使用 wire.Struct 自动注入所有字段
		*wireItem = append(*wireItem, fmt.Sprintf(`wire.Struct(new(%s), "*")`, stName))
	}

	// 添加接口绑定
	for _, itf := range elem.implements {
		var itfName string
		if strings.Contains(itf, ".") {
			itfName = itf
		} else {
			itfName = appendPkg(elem.pkg, itf)
		}
		// 生成 wire.Bind(new(Interface), new(*Implementation))
		*wireItem = append(*wireItem, fmt.Sprintf(`wire.Bind(new(%s), new(*%s))`, itfName, stName))
	}

	// 如果标记为 init，添加到 initElements
	if elem.initWire {
		sc.mu.Lock()
		sc.initElements = append(sc.initElements, *elem)
		sc.mu.Unlock()
	}
}

// createImportSpec 创建导入规范.
func (sc *autoWireSearcher) createImportSpec(elem *element) *ast.ImportSpec {
	imp := &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf(`"%s"`, elem.pkgPath),
		},
	}
	// 如果包名与路径最后一段不同，需要指定别名
	_, last := filepath.Split(elem.pkgPath)
	if last != elem.pkg {
		imp.Name = ast.NewIdent(elem.pkg)
	}
	return imp
}

// writeConfigFile 写入配置文件.
func (sc *autoWireSearcher) writeConfigFile(fileName string, data wireSet, importPkgs []*ast.ImportSpec) error {
	fs := token.NewFileSet()
	src := bytes.NewBuffer(nil)

	// 使用模板生成基础代码
	if err := setTemp.Execute(src, data); err != nil {
		return fmt.Errorf("执行模板失败: %w", err)
	}

	// 解析生成的代码，添加 import 语句
	f, err := parser.ParseFile(fs, "", src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("解析生成的代码失败: %w", err)
	}
	if decl, ok := f.Decls[0].(*ast.GenDecl); ok {
		for _, imp := range importPkgs {
			decl.Specs = append(decl.Specs, imp)
		}
	}

	// 格式化代码
	setDataBuf := &bytes.Buffer{}
	if err := format.Node(setDataBuf, fs, f); err != nil {
		return fmt.Errorf("格式化代码失败: %w", err)
	}

	// 处理 import 并写入文件
	return importAndWrite(fileName, setDataBuf.Bytes())
}

// writeSets 生成汇总文件和初始化入口文件
// 生成两个文件：
// 1. autowire_sets.go - 包含所有 Set 的汇总
// 2. wire.gen.go - 包含初始化函数入口.
//
//nolint:funlen
func (sc *autoWireSearcher) writeSets() error {
	if len(sc.sets) == 0 {
		return nil
	}

	// 任务1: 生成 autowire_sets.go
	sc.wg.Go(func() error {
		slices.Sort(sc.sets)

		fileName := filepath.Join(sc.genPath, filePrefix+"_sets.go")
		bf := bytes.NewBuffer(nil)

		// 创建一个包含所有 Set 的大 Set
		set := wireSet{
			Package: sc.pkg,
			SetName: "Sets",
			Items:   []string{strings.Join(sc.sets, ",\n\t")},
		}

		// 使用模板生成代码
		if err := setTemp.Execute(bf, &set); err != nil {
			return fmt.Errorf("执行模板失败: %w", err)
		}

		// 写入文件
		return importAndWrite(fileName, bf.Bytes())
	})

	// 任务2: 生成 wire.gen.go（初始化函数入口）
	sc.wg.Go(func() error {
		// 如果没有 init 元素或未指定 initWire，跳过
		if len(sc.initElements) == 0 || len(sc.initWire) == 0 {
			return nil
		}

		// 按名称排序，保证生成的代码顺序稳定
		slices.SortFunc(sc.initElements, func(a, b element) int {
			return strings.Compare(a.name, b.name)
		})

		// 生成文件头部
		inits := []string{fmt.Sprintf(initTemplateHead, sc.pkg)}

		// 收集所有配置参数
		configs := make([]string, 0, len(sc.configElements))
		slices.SortFunc(sc.configElements, func(a, b element) int {
			return strings.Compare(a.name, b.name)
		})

		// 为每个配置生成参数：c0 *Config, c1 *AnotherConfig
		for i, c := range sc.configElements {
			configs = append(configs, fmt.Sprintf(`c%d *%s`, i, appendPkg(c.pkg, c.name)))
		}

		paramConfig := strings.Join(configs, ",")

		// 生成初始化函数
		if len(sc.initWire) == 1 && sc.initWire[0] == "*" {
			// 为所有 init 元素生成初始化函数
			for _, w := range sc.initElements {
				inits = append(inits, fmt.Sprintf(initItemTemplate, w.name, paramConfig, "*"+appendPkg(w.pkg, w.name)))
			}
		} else {
			// 只为指定的类型生成初始化函数
			for _, i := range sc.initWire {
				sp := strings.Split(i, ".")
				inits = append(inits, fmt.Sprintf(initItemTemplate, sp[len(sp)-1], paramConfig, i))
			}
		}

		// 写入 wire.gen.go
		wireGenData := strings.Join(inits, "\n")
		return importAndWrite(filepath.Join(sc.genPath, "wire.gen.go"), []byte(wireGenData))
	})

	return sc.wg.Wait()
}
