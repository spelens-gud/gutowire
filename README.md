# Go template

## 环境设置

### 安装 Go

登录 Go 官网下载go二进制包`https://go.dev/doc/install`

```bash
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.25.3.linux-amd64.tar.gz
```

### 安装 gogen

gogen 是一个用于生成项目模板的工具。它可以使用已有的 github repo 作为模版生成新的项目。

```bash
go install golang.org/x/tools/cmd/gonew@latest
```

```bash
gonew github.com/Anniext/go_demo new.example.com/myapp
```

### 安装 pre-commit

pre-commit 是一个代码检查工具，可以在提交代码前进行代码检查。

```bash
pipx install pre-commit
```

安装成功后运行 `pre-commit install` 即可。

### 安装 goimports

goimports 是一个用于格式化 Go 代码的工具。

```bash
go install golang.org/x/tools/cmd/goimports@latest
```

### 安装 golangci-lint

golangci-lint 是一个用于检查 Go 代码的工具。

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

最新版支持go1.25需要golangci-lint@2.5

### 安装 govulncheck

govulncheck 是一个用于检查 Go 代码漏洞的工具。

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
```

### 安装 typos

typos 是一个拼写检查工具。

```bash
cargo install typos-cli
```

### 安装 git cliff

git cliff 是一个生成 changelog 的工具。

```bash
cargo install git-cliff
```

读取cliff.toml 生成更变日志

```bash
git cliff --output CHANGELOG.md
```
