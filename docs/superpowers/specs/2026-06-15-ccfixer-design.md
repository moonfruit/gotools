# ccfixer 设计文档

日期：2026-06-15

## 1. 背景与问题

新版 Claude Code 在发起 Anthropic Messages API 请求时，会在 `messages` 数组**中间**插入
`role: "system"` 的消息。这是 Anthropic 的正式特性 **mid-conversation system message**
（beta header `mid-conversation-system-2026-04-07`，Opus 4.7 起支持），用于在会话进行中
下发带「操作者权威」的指令（模式切换、补充上下文等），同时避免改顶层 `system` 字段而
让 prompt cache 失效。

但部分第三方中转商 / 自建推理服务用严格 schema 校验请求体，尚未实现该 beta，于是拒绝
数组里出现 `system` 角色，报错形如：

```
{'type': 'literal_error', 'loc': ('body', 'messages', 1, 'role'),
 'msg': "Input should be 'user' or 'assistant'", 'input': 'system',
 'ctx': {'expected': "'user' or 'assistant'"}}
```

本工具是一个**透明反向代理**，拦截 Claude Code 的请求，把这类 `system` 消息就地改写成
中转商能接受的形式后再转发，其余内容（headers、path、响应流）原样透传。

## 2. 部署链路与约束

```
Claude Code ──POST /v1/messages──▶ ccfixer ──(改写 body)──▶ 上游中转商
                                      │
                          仅解析+改写请求 JSON，
                          响应 body 直接透传（天然支持流式 SSE）
```

目标上游示例：以 vLLM 运行 Qwen3.6 的 OpenAI/Anthropic 兼容服务。

**关键约束——保住 vLLM prefix cache：** vLLM 的 Automatic Prefix Caching 是 token 级
前缀匹配，前缀一旦变，后面全部失效；而 chat template 会把 `system` 渲染到序列最前面。
若把逐轮变化的 mid-conversation system 内容**上提到顶层 `system`**，等于每轮都改动序列
开头 → 整条缓存前缀失效 → 几乎全量 cache miss。因此**必须把该消息留在原位**（并入相邻
user 消息），只让序列尾部变化，长而稳的前缀得以持续命中缓存。这是选定改写策略的根本原因。

## 3. 改写策略（核心）

### 3.1 总则

遍历请求体 `messages` 数组，把每个 `role == "system"` 的元素**并入相邻的 user 消息**，
然后从数组中删除该元素。理由（相较其它方案）：

- 不上提到顶层 `system` → 保住 vLLM prefix cache（见 §2）。
- 并入 user → 同时通过 role 校验与 user/assistant 交替校验。
- 用 `<system-reminder>` 包裹 → 见 §3.3。

### 3.2 并入目标的选择规则

对每个 system 元素，按以下优先级确定并入目标：

1. **紧邻的前一条 user 消息**（API 约束下 mid-conversation system 基本都跟在 user 后面，
   这是主路径）。
2. 若前一条不是 user（罕见，如跟在带 server tool result 的 assistant 之后）→ **紧邻的
   后一条 user 消息**。
3. 两侧都没有可并入的 user → **兜底**：直接把该元素的 `role` 改为 `"user"`，保证至少
   通过 role 校验。

### 3.3 内容包裹与拼接

并入时，把 system 文本用 `<system-reminder>...</system-reminder>` 包裹后再追加进目标
user 消息。

`<system-reminder>` 的作用（针对 Qwen 链路如实说明）：

- **不**为 Qwen 带来「系统权威 / 防注入」语义——那是 Claude 被专门训练过的约定，Qwen 没有。
- 提供**结构/来源分隔**：把注入指令与用户原话用清晰边界隔开。
- 提供**可调试性与一致性**：便于排查；且 Claude Code 本身就大量用 `<system-reminder>`
  包裹注入上下文，保持同一风格。

content 有两种形态，需都兼容：

- **string**：把 `<system-reminder>` 包裹后的文本拼接到原字符串末尾（之间加换行分隔）。
- **block 数组**：在数组末尾追加一个 `{"type": "text", "text": "<system-reminder>...</system-reminder>"}` 文本块。

system 消息自身的 content 同样可能是 string 或 block 数组：

- string → 直接取用。
- block 数组 → 把其中所有 `type == "text"` 块的文本按顺序用换行拼接；非文本块（极少）忽略。

## 4. 代理行为

- 实现：标准库 `net/http` + `httputil.ReverseProxy`，在转发前读取并改写请求 body，重算
  `Content-Length`。
- **只改写**满足全部条件的请求：`POST` + path 含 `/v1/messages` + body 是合法 JSON 且含
  `messages` 数组。其它请求一律原样透传。
- **Fail-open**：body 读取/解析失败、或结构不符预期 → 不改写，原样转发，绝不因改写逻辑
  报错而中断 Claude Code。
- **headers 全透传**：`Authorization` / `x-api-key` / `anthropic-version` /
  `anthropic-beta` 等均不改动。
- **响应透传**：响应 body 直接透传，天然支持流式 SSE（代理不解析响应）。

## 5. 配置（cobra flags）

| flag | 说明 | 默认值 |
|---|---|---|
| `-l, --listen` | 监听地址 | `127.0.0.1:8787` |
| `-u, --upstream` | 上游 base URL（**必填**），如 `https://your-relay.example.com` | 无 |
| `-v, --verbose` | 打印每次改写了几条 system 消息，便于排查 | `false` |

转发时保留原始请求的 path 与 query，只替换 scheme+host 为 `--upstream`。

## 6. 仓库布局

遵循仓库既有约定（薄入口、逻辑入 internal、工厂函数返回 root cmd）：

```
cmd/ccfixer/
  main.go        # os.Exit(Execute()) 之类
  root.go        # newRootCmd() *cobra.Command；flag 定义、起 HTTP 服务、装配 ReverseProxy
  root_test.go   # 用 httptest 假上游驱动整条代理链路的集成测试
internal/ccfixer/
  transform.go   # 纯函数 Transform(body []byte) (out []byte, n int, err error)
  transform_test.go
```

- 核心改写抽成纯函数 `Transform`，输入/输出均为 `[]byte`，返回改写条数（供 verbose 与
  测试用），不依赖 HTTP，便于单测。
- root cmd 用 `newRootCmd()` 工厂函数返回，避免包级单例。

## 7. 测试

**`internal/ccfixer`（单元，表驱动）覆盖：**

- system 跟在 user 之后（主路径），content 为 string。
- system 跟在 user 之后，content 为 block 数组。
- 目标 user 的 content 为 string / 为 block 数组两种形态。
- system 跟在 assistant 之后 → 并入后一条 user。
- 多条 system 混在数组各处。
- 两侧无 user → 兜底改 role 为 user。
- 无 `messages` 字段 / `messages` 非数组 → 原样返回。
- 非法 JSON → 原样返回、不报错。
- 确认 `<system-reminder>` 包裹正确出现在结果中。

**`cmd/ccfixer`（集成）覆盖：**

- 用 `httptest.Server` 起假上游，断言上游收到的 body 中已无 `system` 角色、且包含
  `<system-reminder>`。
- 断言响应（含分块/流式）能原样透传回客户端。
- 断言非 `/v1/messages` 请求原样透传。

## 8. 范围控制（YAGNI）

- 只实现这一个改写规则。代码结构（`Transform` 纯函数）为未来追加规则留有余地，但**现在
  不实现**任何配置化的通用改写引擎。
- 不做鉴权、限流、缓存、日志持久化等——仅做请求体改写 + 透传。
