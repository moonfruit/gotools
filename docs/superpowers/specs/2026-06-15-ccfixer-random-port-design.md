# ccfixer 随机端口与 shell 集成 设计文档

日期：2026-06-15

## 1. 背景与目标

`ccfixer`（透明反向代理，见 `2026-06-15-ccfixer-design.md`）当前用固定监听地址
（`-l`，默认 `127.0.0.1:8787`）。本次增强：

1. 支持**监听本地随机可用端口**——把端口交给 OS 自动分配。
2. 把选定的监听地址以**便于 shell 集成**的方式返回，方便脚本拿到端口/URL 后把
   Claude Code 指向该代理（`ANTHROPIC_BASE_URL`）。

## 2. 触发方式

**复用现有 `-l` flag，端口写 `0`**——这是 Unix 惯例（端口 0 = 让内核分配空闲端口）。

- `-l :0` 或 `-l 127.0.0.1:0` → 随机端口。
- 不新增 `--random` / `--port-file` 等 flag（YAGNI）。
- 默认值仍为 `127.0.0.1:8787`，固定端口行为不变。

## 3. 返回方式（shell 集成）

绑定成功后：

- 把解析出的 **base URL**（如 `http://127.0.0.1:54321`）作为**单独一行打印到 stdout**
  （机器可读，便于脚本读取）。
- 把人类可读 banner `ccfixer listening on <addr>, forwarding to <upstream>` 打印到
  **stderr**（原实现打在 stdout，本次改到 stderr 以分离机器输出与日志）。

这个「URL→stdout、banner→stderr」对**固定端口和随机端口一视同仁**，输出可预测。

### Shell 集成惯用法（写入 README）

服务常驻阻塞，`$(...)` 会卡住，推荐进程替换 + 读首行：

```bash
exec 3< <(ccfixer -u https://relay.example.com -l :0)
read -r ANTHROPIC_BASE_URL <&3      # 取到 http://127.0.0.1:<随机端口>
export ANTHROPIC_BASE_URL
# ccfixer 在后台继续 serve；shell 退出时自动收尾
```

## 4. 实现方案

### 4.1 绑定流程（`runRoot`）

从 `ListenAndServe()` 改为先显式绑定再 serve：

1. `ln, err := net.Listen("tcp", opts.listen)`；绑定失败（端口被占、地址非法）直接返回
   错误。**先绑定再打印**——绑定失败时不再先打 banner 后报错（修掉既有小瑕疵）。
2. `port := ln.Addr().(*net.TCPAddr).Port` 取真实端口。
3. `baseURL, err := resolveBaseURL(opts.listen, port)` 计算 URL。
4. `fmt.Fprintln(cmd.OutOrStdout(), baseURL)` —— URL 到 stdout。
5. `fmt.Fprintf(cmd.ErrOrStderr(), "ccfixer listening on %s, forwarding to %s\n", ln.Addr(), opts.upstream)` —— banner 到 stderr。
6. `server := &http.Server{Handler: proxy}`；`return server.Serve(ln)`（不再设 `Addr`）。

### 4.2 URL host 推导（纯函数）

```
func resolveBaseURL(listenAddr string, port int) (string, error)
```

- `host, _, err := net.SplitHostPort(listenAddr)`；非法地址返回 err。
- host 为空（如 `:0`），或为未指定地址（`0.0.0.0` / `::`，用 `net.ParseIP(host).IsUnspecified()`
  判断）→ 替换为 `127.0.0.1`，保证 URL 可直接访问。
- 其余 host（`localhost`、具体 IP）保留原样。
- 返回 `"http://" + net.JoinHostPort(host, strconv.Itoa(port))`。

该函数不触碰 socket，纯字符串/IP 逻辑，便于表驱动单测。

## 5. 测试

**`resolveBaseURL` 表驱动单测（`cmd/ccfixer/root_test.go`）：**

- `":0"`, port=54321 → `http://127.0.0.1:54321`
- `"127.0.0.1:0"`, port=p → `http://127.0.0.1:p`
- `"0.0.0.0:0"`, port=p → `http://127.0.0.1:p`（未指定地址替换）
- `"[::]:0"`, port=p → `http://127.0.0.1:p`（未指定 IPv6 替换）
- `"localhost:0"`, port=p → `http://localhost:p`（主机名保留）
- 非法地址（如 `"bogus"`）→ 返回 error

**真实绑定测试：**

- `net.Listen("tcp", "127.0.0.1:0")` 取实际端口，断言 `resolveBaseURL("127.0.0.1:0", port)`
  == `http://127.0.0.1:<port>` 且 port > 0；用完关闭 listener。

**回归：** 现有代理集成测试（`TestProxyRewritesSystemMessage` 等）不受影响——它们直接驱动
`newProxy`，不依赖 `runRoot` 的 stdout/stderr 输出。

## 6. 文档

- README 的 `ccfixer` 小节：补充 `-l :0` 随机端口用法、stdout(URL)/stderr(banner) 约定，
  以及上面的 shell 集成惯用法片段。
- CLAUDE.md 现有工具表 `ccfixer` 行：补一句「`-l :0` 监听随机空闲端口，选定 URL 打到 stdout」。

## 7. 范围控制（YAGNI）

- 不新增 flag（不做 `--random`、不做 `--port-file`）。
- 不改写改写逻辑、代理逻辑，仅改 `runRoot` 的绑定/输出与新增 `resolveBaseURL`。
