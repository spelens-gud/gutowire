package iwanta

// 快速原型开发工具模块
// 提供 IWantA 魔法函数，用于快速获取任何类型的实例
// 适用于测试和原型开发场景

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/spelens-gud/gutowire/internal/config"
	"github.com/spelens-gud/gutowire/internal/parser"
	"github.com/spelens-gud/gutowire/internal/runner"
	"github.com/stoewer/go-strcase"
)

const (
	// thisIsYourTemplate 生成的辅助函数模板
	// 用于包装 Initialize 函数，使其更易用.
	thisIsYourTemplate = `
func thisIsYour%s(res *%s,%s) (err error, cleanup func()) {
	*res, cleanup, err = %s
	return
}
`
)

// regexpCall 用于匹配 IWantA 调用的正则表达式.
var regexpCall = regexp.MustCompile(`gutowire\.IWantA\(&([a-zA-Z]+).*?\)`)

// iwantA struct    功能的内部状态.
type iwantA struct {
	wantInputIdent     string   // 输入参数的标识符
	thisIsYourFuncName string   // 生成的函数名称
	callFileLines      []string // 调用文件的所有行
	callLine           int      // 调用所在的行号
	callFile           string   // 调用文件的路径
}

// initWantArgIdent method    初始化输入参数标识符
// 从调用代码中提取变量名.
func (iw *iwantA) initWantArgIdent() {
	callLineStr := regexpCall.FindAllStringSubmatch(strings.TrimSpace(iw.callFileLines[iw.callLine-1]), -1)
	for i := range callLineStr {
		if len(callLineStr[i]) == 2 {
			iw.wantInputIdent = callLineStr[i][1]
			break
		}
	}

	// 重写调用代码，将 IWantA 替换为 thisIsYour
	if iw.wantInputIdent == "" {
		iw.wantInputIdent = "nil"
	} else {
		iw.wantInputIdent = "&" + iw.wantInputIdent
	}
}

// IWantA function    魔法函数：快速获取任何类型的实例
// 这是一个特殊的开发辅助工具，用于快速原型开发和测试
//
// 工作原理：
// 1. 通过反射获取想要的类型信息
// 2. 自动运行 AutoWire 生成依赖注入代码
// 3. 生成 thisIsYour<Type> 辅助函数
// 4. 修改调用代码，替换 IWantA 为生成的函数
// 5. 程序退出，让开发者重新运行
//
// 使用示例：
//
//	var zoo Zoo
//	gutowire.IWantA(&zoo)  // 第一次运行会生成代码并退出
//	// 重新运行后，zoo 就会被正确初始化
//
// in: 指向想要类型的指针
// searchDepDirs: 可选的依赖搜索目录.
func IWantA(in interface{}, searchDepDirs ...string) (_ struct{}) {
	// 如果未指定搜索目录，使用模块根目录
	if len(searchDepDirs) == 0 {
		modPath := parser.GetGoModDir()
		if len(modPath) > 0 {
			searchDepDirs = append(searchDepDirs, modPath)
		}
	}

	// 获取调用位置信息
	_, callFile, callLine, ok := runtime.Caller(1)
	if !ok {
		panic("无法获取调用路径")
	}

	// 读取调用文件内容
	callFileData, err := os.ReadFile(callFile)
	if err != nil {
		panic(fmt.Sprintf("读取调用文件失败: %v", err))
	}

	iw := &iwantA{
		callFile:      callFile,
		callLine:      callLine,
		callFileLines: strings.Split(string(callFileData), "\n"),
	}

	// 提取输入参数标识符
	iw.initWantArgIdent()

	// 生成 wire.go
	var (
		wantTypeVar string
		genSuccess  bool

		rType       = reflect.TypeOf(in).Elem()
		modeBase, _ = parser.GetModBase()
		callPkgPath = parser.GetPkgPath(callFile, modeBase)
	)

	// 确定类型的完整名称
	if rType.PkgPath() == callPkgPath {
		// 同一个包，只需要类型名
		wantTypeVar = rType.Name()
	} else {
		// 不同包，需要完整路径
		wantTypeVar = rType.String()
	}

	wantTypeName := strcase.SnakeCase(strings.Replace(strings.Replace(wantTypeVar, "_", "", -1), ".", "_", -1))
	genPath := filepath.Dir(callFile)
	wireOpt := make([]config.Option, 0)

	iw.thisIsYourFuncName = strcase.UpperCamelCase(wantTypeName)

	// 清理临时文件
	defer func() {
		iw.cleanIWantATemp(callFile)
		if genSuccess {
			// 生成成功后退出，让开发者重新运行
			os.Exit(0)
		}
	}()

	// 配置搜索路径
	wireOpt = append(wireOpt, parser.Map(searchDepDirs, func(s string) config.Option {
		return config.WithSearchPath(s)
	})...)

	// 指定要初始化的类型
	wireOpt = append(wireOpt, config.InitStruct(strings.TrimPrefix(wantTypeVar, "*")))

	// 运行 autowire 生成代码
	if err := runner.RunAutoWire(genPath, wireOpt...); err != nil {
		panic(err)
	}

	// 生成初始化函数
	args, err := iw.writeInitFile(wantTypeVar, wantTypeName)
	if err != nil {
		panic(err)
	}

	// 更新调用文件
	if err = iw.updateCallFile(args); err != nil {
		panic(err)
	}

	genSuccess = true
	return struct{}{}
}

// updateCallFile method    更新调用文件，将 IWantA 替换为生成的函数.
func (iw *iwantA) updateCallFile(configArgs []string) (err error) {
	callLine := strings.TrimSpace(iw.callFileLines[iw.callLine-1])
	callArgs := strings.Join(append([]string{iw.wantInputIdent}, configArgs...), ",")
	assignStr := fmt.Sprintf("_, _ = thisIsYour%s(%s)", iw.thisIsYourFuncName, callArgs)

	// 如果原来是 var 声明，保留 var 关键字
	if strings.HasPrefix(callLine, "var ") {
		assignStr = "var " + assignStr
	}

	// 注释掉原来的 IWantA 调用
	iw.callFileLines[iw.callLine-1] = "// " + callLine
	// 插入新的调用代码
	iw.callFileLines = append(iw.callFileLines[:iw.callLine],
		append([]string{assignStr}, iw.callFileLines[iw.callLine:]...)...)
	return parser.ImportAndWrite(iw.callFile, []byte(strings.Join(iw.callFileLines, "\n")))
}

// regexpInitMethod 用于匹配 Initialize 函数的正则表达式.
var regexpInitMethod = regexp.MustCompile(`Initialize(.+?)\((.*?)\)`)

// writeInitFile method    生成初始化辅助文件
// 读取 wire_gen.go，提取 Initialize 函数，生成 thisIsYour 包装函数.
func (iw *iwantA) writeInitFile(wantVar, name string) (args []string, err error) {
	genPath := filepath.Dir(iw.callFile)
	//nolint:gosec
	initFileData, err := os.ReadFile(filepath.Join(genPath, "wire_gen.go"))
	if err != nil {
		return nil, fmt.Errorf("读取 wire_gen.go 失败: %w", err)
	}

	// 从 wire_gen.go 中提取 Initialize 函数签名
	var call string
	ret := regexpInitMethod.FindStringSubmatch(string(initFileData))
	if len(ret) >= 2 {
		var argsVar []string
		if len(ret) > 2 {
			// 解析函数参数
			params := parser.Filter(strings.Split(ret[2], ","), func(sp string) bool {
				return len(strings.SplitN(sp, " ", 2)) == 2
			})
			for _, sp := range params {
				spp := strings.SplitN(sp, " ", 2)
				// 为每个参数生成零值
				args = append(args, "&"+strings.TrimPrefix(spp[1], "*")+"{}")
				argsVar = append(argsVar, spp[0])
			}
		}
		// 构造函数调用
		call = fmt.Sprintf(`Initialize%s(%s)`, ret[1], strings.Join(argsVar, ","))
	} else {
		err = errors.New("invalid init file")
		return nil, err
	}

	// 生成文件名
	filename := strcase.SnakeCase(name) + "_init"
	if strings.HasSuffix(iw.callFile, "_test.go") {
		filename += "_test"
	}
	filename += ".go"

	// 生成 thisIsYour 函数
	initFileData = append(initFileData, fmt.Sprintf(thisIsYourTemplate, iw.thisIsYourFuncName, wantVar, ret[2], call)...)
	initFileName := filepath.Join(genPath, filename)
	if err = parser.ImportAndWrite(initFileName, initFileData); err != nil {
		return nil, fmt.Errorf("写入初始化文件失败: %w", err)
	}
	return args, nil
}

// cleanIWantATemp method    清理 IWantA 生成的临时文件
// 删除所有 autowire 相关文件和 wire 生成文件.
func (iw *iwantA) cleanIWantATemp(f string) {
	dir := filepath.Dir(f)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "autowire") ||
			name == "wire.gen.go" ||
			name == "wire_gen.go" {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}
