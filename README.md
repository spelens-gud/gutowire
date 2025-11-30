# GuToWire

一个基于 Google Wire 的 Go 依赖注入代码生成工具，通过注解自动生成依赖注入配置。

## 特性

- 🚀 **注解驱动**：使用 `@autowire` 注解标记需要注入的组件
- 🔧 **自动扫描**：递归扫描项目目录，自动发现所有标记的组件
- 📦 **分组管理**：支持将组件分组到不同的 Set 中
- 🔌 **接口绑定**：自动识别接口实现关系
- ⚙️ **配置注入**：支持配置结构体字段级别的注入
- 🎯 **快速原型**：提供 `IWantA` 魔法函数用于快速开发

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
```

## 示例

查看 `examples/` 目录获取完整示例。

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
