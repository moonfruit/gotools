# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 仓库目标

本仓库用于以 Go 编写一系列**独立的小工具**（CLI / utilities），属于工具集 monorepo 性质。每个工具都是 `cmd/<name>` 下的独立 `package main`，共享逻辑放在 `internal/<pkg>`。

## 工具链

- Go: `go1.26.3 darwin/arm64`（`go.mod` 写为 `go 1.26`）
- 模块路径：`github.com/moonfruit/gotools`
- 模块缓存：`/Users/moon/Workspace.localized/go/pkg/mod`
- `GOPATH`：`/Users/moon/Workspace.localized/go`

## 仓库布局

```
gotools/
├── go.mod, go.sum                # module github.com/moonfruit/gotools
├── bin/                          # 本地构建产物（已在 .gitignore）
├── cmd/<tool>/                   # 每个工具一个目录，package main
└── internal/<tool>/              # 每个工具对应的业务逻辑，可独立测试
```

## 常用命令

```bash
# 构建
go build ./...                                  # 全部
go build -o bin/<tool> ./cmd/<tool>             # 单个工具

# 测试
go test ./...                                   # 全部
go test ./internal/<pkg>                        # 单个包
go test -run TestXxx ./internal/<pkg>           # 单个测试
go test -race ./...                             # race detector

# 静态检查
go vet ./...
gofmt -l .                                      # 列出未格式化文件
go mod tidy                                     # 整理依赖
```

## 写代码时的约定

- **工具入口要薄**：`cmd/<tool>/main.go` 只做 `os.Exit(Execute())` 之类的事；CLI 定义（flag、Args 校验、I/O 选路）放 `cmd/<tool>/root.go`；业务逻辑全部抽到 `internal/<tool>/` 便于单测。
- **工具间互不 import**：共享逻辑通过 `internal/` 提取；不要从一个工具 import 另一个工具。
- **第三方依赖谨慎引入**：优先标准库。已引入的：
  - `github.com/spf13/cobra` — 仅用于命令行解析（提供 shell 补全的 `completion` 子命令）。
  - `github.com/mattn/go-runewidth` — 计算字符的终端显示宽度（East Asian Width / 组合字符），`wcwidth` 工具使用。
- **工具的 root cmd 用工厂函数返回**：例如 `newRootCmd() *cobra.Command`，避免包级单例污染测试。
- **测试组织**：单元测试与代码同包同目录（`*_test.go`）；CLI 集成测试也放对应 `cmd/<tool>/` 下，通过 `cmd.SetIn/SetOut/SetErr/SetArgs` 驱动。

## 现有工具

| 工具 | 入口 | 内核包 | 说明 |
|---|---|---|---|
| `uhsort` | `cmd/uhsort/` | `internal/uhsort/` | 按 host→port→user→rest 排序 `user@host[:port]` 列表，支持去重与计数，支持 stdin/stdout、`-o` 与 `-i` 原地替换。 |
| `wcwidth` | `cmd/wcwidth/` | `internal/wcwidth/` | 计算 UTF-8 文本的终端显示宽度（CJK/全角=2、组合字符=0）。有参数时逐参数输出宽度，无参数时读 stdin 逐行输出；`-E`/`-N` 控制 East Asian Ambiguous 字符按宽 2 / 宽 1。 |
