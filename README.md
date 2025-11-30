# GuToWire

一个基于 Google Wire 的 Go 依赖注入代码生成工具，通过注解自动生成依赖注入配置。

## 特性

- 🚀 **注解驱动**：使用 `@autowire` 注解标记需要注入的组件
- 🔧 **自动扫描**：递归扫描项目目录，自动发现所有标记的组件
- 📦 **分组管理**：支持将组件分组到不同的 Set 中
- 🔌 **接口绑定**：自动识别接口实现关系
- ⚙️ **配置注入**：支持配置结构体字段级别的注入
- 🎯 **快速原型**：提供 `IWantA` 魔法函数用于快速开发
- ⚡ **高性能**：并发扫描，智能缓存，性能提升 50-80%
- 👀 **Watch 模式**：自动监听文件变化并重新生成
- 📝 **配置文件**：支持 YAML 配置文件管理项目设置

## 快速开始

### 安装

```bash
go install github.com/spelens-gud/gutowire@latest
```

### 基本用法

1. 在你的结构体或构造函数上添加 `@autowire` 注解：

```go
// @autowire(set=animals)
type Dog struct {
    Name string
}

// @autowire(set=animals)
func NewCat() *Cat {
    return &Cat{}
}
```

2. 运行 gutowire 生成 Wire 配置：

```bash
gutowire ./path/to/your/package
```

3. 运行 wire 生成最终代码：

```bash
cd ./path/to/your/package && wire
```

### 注解语法

#### 基础用法

```go
// @autowire(set=animals)
type Dog struct {}
```

#### 接口绑定

```go
// @autowire(set=animals,Animal)
type Dog struct {}
```

#### 自定义构造函数

```go
// @autowire(set=animals,new=CustomConstructor)
type Dog struct {}

func CustomConstructor() *Dog {
    return &Dog{}
}
```

#### 初始化入口

```go
// @autowire.init(set=zoo)
type Zoo struct {
    Animals []Animal
}
```

#### 配置注入

```go
// @autowire.config(set=config)
type Config struct {
    Host string
    Port int
}
```

## 命令行选项

```bash
gutowire [flags] <生成路径>

Flags:
  -w, --wire_path string   Wire 配置文件生成路径
  -s, --scope string       依赖搜索范围(目录路径)，不填则全局搜索
  -p, --pkg string         生成文件的包名
  --watch                  启用 Watch 模式，自动监听文件变化
  --config string          指定配置文件路径（默认 .gutowire.yaml）
  --init                   生成默认配置文件
  --no-cache              禁用文件缓存
```

## 高级功能

### Watch 模式

自动监听文件变化并重新生成代码，适合开发阶段使用：

```bash
# 启动 watch 模式
gutowire --watch ./wire

# 修改代码后会自动重新生成
```

### 配置文件

使用 YAML 配置文件管理项目设置：

```bash
# 生成默认配置文件
gutowire --init

# 使用配置文件
gutowire --config=.gutowire.yaml
```

配置文件示例（`.gutowire.yaml`）：

```yaml
# 基础配置
search_path: ./ # 依赖搜索路径
output_path: ./wire # 输出路径
package: wire # 生成文件的包名

# 初始化配置
init_types: # 需要生成初始化函数的类型
  - App
  - Server

# 性能配置
enable_cache: true # 启用缓存（默认 true）
parallel: 0 # 并发数，0 表示自动检测 CPU 核心数

# 高级配置
exclude_dirs: # 排除的目录（可自定义）
  - vendor
  - testdata
  - .git

# Watch 模式配置
watch: false # 是否启用 watch 模式
watch_ignore: # watch 模式忽略的文件模式
  - "*.gen.go"
  - "wire_gen.go"
```

## 示例

查看 `examples/` 目录获取完整示例。

## 性能优化

GuToWire v2.0 进行了全面的性能优化：

- **并发扫描**：真正的并发文件处理
- **智能检查**：快速检查文件是否包含注解
- **智能缓存**：缓存已解析的文件，避免重复解析
- **路径缓存**：避免重复计算包路径
- **总体提升**：性能提升

### 缓存功能

GuToWire 支持智能缓存，大幅提升重复生成的性能：

```bash
# 默认启用缓存
gutowire ./wire

# 禁用缓存
gutowire --no-cache ./wire

# 通过配置文件控制
# .gutowire.yaml
enable_cache: true
```

缓存文件保存在生成目录下的 `.gutowire.cache`，通过文件修改时间和内容哈希判断文件是否变化。

### 性能对比

| 项目规模           | 优化前 | 优化后 | 提升 |
|----------------|-----|-----|----|
| 小型（<100 文件）    | 3s  | 1s  | 3x |
| 中型（100-500 文件） | 15s | 5s  | 3x |
| 大型（>1000 文件）   | 60s | 15s | 4x |

## 功能特性详解

### 缓存系统

GuToWire 实现了完整的缓存系统，显著提升重复生成的性能：

**工作原理**：

- 缓存文件保存在生成目录的 `.gutowire.cache`
- 通过文件修改时间和内容哈希判断文件是否变化
- 未修改的文件直接使用缓存，跳过解析过程

**使用方式**：

```bash
# 默认启用缓存
gutowire ./wire

# 禁用缓存
gutowire --no-cache ./wire

# 配置文件控制
# .gutowire.yaml
enable_cache: true
```

**性能提升**：

- 首次运行：正常解析所有文件
- 后续运行：仅解析修改过的文件

### 自定义排除目录

支持通过配置文件自定义需要排除的目录：

```yaml
exclude_dirs:
  - vendor # 默认排除
  - testdata # 默认排除
  - .git # 默认排除
  - node_modules # 自定义添加
  - dist # 自定义添加
```

### 错误提示

提供详细的错误信息和解决建议：

- **文件不存在**：提示检查路径和文件是否存在
- **解析失败**：显示详细的错误位置和原因
- **循环依赖**：警告并提供解决方案
- **Wire 错误**：格式化 Wire 输出，提供针对性建议

### Watch 模式

自动监听文件变化，实时重新生成代码：

```bash
# 启动 watch 模式
gutowire --watch ./wire

# 或通过配置文件
# .gutowire.yaml
watch: true
```

**特性**：

- 自动监听 `.go` 文件变化
- 防抖机制，避免频繁触发
- 忽略生成的文件（`*.gen.go`, `wire_gen.go`）
- 支持自定义忽略模式

## 更新日志

### v2.1 (2025-12-01)

**功能完善**：

- ✅ 缓存功能完全接入：支持命令行和配置文件控制
- ✅ 错误提示接入：在关键位置使用错误
- ✅ 配置文件排除目录：支持自定义排除目录
- ✅ 并发安全改进：修复循环变量捕获问题
- ✅ 代码结构优化：改进错误处理和缓存机制

**性能提升**：

| 场景   | 无缓存    | 有缓存    | 提升     |
|------|--------|--------|--------|
| 扫描文件 | 17 个   | 17 个   | -      |
| 解析注解 | 17 个   | 0 个    | 100%   |
| 生成时间 | ~1-2 秒 | ~0.5 秒 | 50-60% |

### v2.0 (2025-12-01)

**性能优化**：

- 修复并发安全问题，实现真正的并发处理
- 添加文件快速检查，避免不必要的完整读取
- 优化包路径计算，减少重复计算
- 改进错误处理和日志记录

**新增功能**：

- Watch 模式：自动监听文件变化
- 配置文件支持：YAML 配置管理
- 友好的错误提示：详细的错误信息和解决建议

**代码质量**：

- 添加 30+ 单元测试
- golangci-lint 检查通过（0 issues）
- 测试覆盖率：config 95.5%, parser 53.5%

## 开发环境设置

### 必需工具

#### 安装 Go

从 Go 官网下载二进制包：`https://go.dev/doc/install`

```bash
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.25.3.linux-amd64.tar.gz
```

#### 安装 Google Wire

```bash
go install github.com/google/wire/cmd/wire@latest
```

## 许可证

查看 LICENSE 文件了解详情。
