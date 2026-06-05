# wcwidth 设计文档

日期：2026-06-05

## 目标

新增 `wcwidth` 工具：计算 UTF-8 文本在终端中的**显示宽度**。

- 半角字符（ASCII 等）宽 1
- CJK / 全角字符宽 2
- 组合字符（如 U+0301 COMBINING ACUTE ACCENT）宽 0
- 控制字符等按 go-runewidth 规则处理

类比 C 的 `wcwidth(3)` / `wcswidth(3)`。

## 依赖

引入 `github.com/mattn/go-runewidth`，用 `runewidth.StringWidth()` 计算。

理由：Go 标准库没有 wcwidth 等价物，East Asian Width 与组合字符判定需要 Unicode 数据表；go-runewidth 是社区事实标准，cobra 生态已广泛使用，正确处理 CJK 宽字符、组合字符（0 宽）、emoji、控制字符与 East Asian Ambiguous 宽度。

## 仓库布局（沿用既有约定）

```
cmd/wcwidth/main.go        # os.Exit 入口（薄）
cmd/wcwidth/root.go        # cobra 定义、I/O 选路、输出格式化
cmd/wcwidth/root_test.go   # CLI 集成测试
internal/wcwidth/width.go      # 纯逻辑：Width(string, *bool) int
internal/wcwidth/width_test.go # 单元测试
```

## 内核接口（`internal/wcwidth`）

```go
// Width 返回 s 的显示宽度。
// eastAsian 为 nil 时，East Asian Ambiguous 宽度字符按运行环境（$LANG/$LC_*）判定；
// 非 nil 时强制：true => 歧义字符宽 2；false => 歧义字符宽 1。
func Width(s string, eastAsian *bool) int
```

实现：根据 `eastAsian` 选择 `runewidth.Condition`：

- `nil`：使用 `runewidth.DefaultCondition`（其 `EastAsianWidth` 在包初始化时按环境判定）
- `*eastAsian == true`：`&runewidth.Condition{EastAsianWidth: true}`
- `*eastAsian == false`：`&runewidth.Condition{EastAsianWidth: false}`

然后调用 `cond.StringWidth(s)`。

## CLI 行为（`cmd/wcwidth`）

### 输入来源与输出

| 场景 | 行为 |
|---|---|
| 有位置参数 `wcwidth 你好 abc` | 逐参数计算，每个参数一行宽度，输出 `4\n3\n` |
| 无参数 | 读 stdin，**逐行**计算，每行输出该行宽度；空行输出 `0`；行尾换行符不计入宽度 |

### 旗标（East Asian Ambiguous 三态）

| 状态 | 旗标 | 歧义字符宽度 |
|---|---|---|
| 默认 | 无 | 按环境自动判定 |
| 强制宽 | `-E` / `--east-asian` | 2 |
| 强制窄 | `-N` / `--narrow` | 1 |

- `-E` 与 `-N` 互斥：`cmd.MarkFlagsMutuallyExclusive("east-asian", "narrow")`
- 解析为 `*bool` 传给内核：都未给 => `nil`；`-E` => `&true`；`-N` => `&false`

### 错误处理

stdin 读取错误返回 error，`main` 以退出码 1 结束（与 uhsort 一致）。

## 测试要点

内核单测（`internal/wcwidth/width_test.go`）：

- 纯 ASCII：`"abc"` => 3
- CJK：`"你好"` => 4
- 混合：`"你好world"` => 8
- 组合字符：`"é"` => 1
- 空串：`""` => 0
- 含 tab/控制字符：按 go-runewidth 规则验证
- emoji：选取确定宽度的样例验证
- 歧义宽度字符（如 `"±"`）：`Width(s, &true)` => 2，`Width(s, &false)` => 1

CLI 集成测试（`cmd/wcwidth/root_test.go`，通过 `SetIn/SetOut/SetArgs` 驱动）：

- 多参数：`["你好", "abc"]` => `"4\n3\n"`
- stdin 多行（含空行）逐行输出
- `-E` 与 `-N` 对歧义字符的不同结果
- `-E -N` 同时给出 => 报错

## 文档维护

实现完成后，在根 `CLAUDE.md` 的"现有工具"表中追加 `wcwidth` 行。
