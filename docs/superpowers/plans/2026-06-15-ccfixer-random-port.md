# ccfixer 随机端口与 shell 集成 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 `ccfixer` 支持 `-l :0` 监听随机空闲端口，并把选定的 base URL 打到 stdout（人类 banner 改到 stderr），便于 shell 集成。

**Architecture:** 新增纯函数 `resolveBaseURL(listenAddr string, port int) (string, error)` 计算可用的 base URL；把 `runRoot` 从 `http.Server.ListenAndServe()` 改为显式 `net.Listen` + `server.Serve(ln)`，绑定后从 `ln.Addr()` 读真实端口，URL→stdout、banner→stderr。

**Tech Stack:** Go 1.26 标准库（`net`、`net/http`、`strconv`）、`github.com/spf13/cobra`。无新增第三方依赖。

参考设计文档：`docs/superpowers/specs/2026-06-15-ccfixer-random-port-design.md`

---

## File Structure

只改动 `ccfixer` 的 CLI 文件，逻辑保持薄：

```
cmd/ccfixer/root.go        # 新增 resolveBaseURL；改写 runRoot 的绑定与输出
cmd/ccfixer/root_test.go   # resolveBaseURL 表驱动单测 + 真实绑定测 + runRoot 错误路径回归测
README.md                  # ccfixer 小节补随机端口 + shell 集成
CLAUDE.md                  # 现有工具表 ccfixer 行补一句
```

`resolveBaseURL` 是纯字符串/IP 逻辑、不碰 socket，单一职责、可独立单测；`runRoot` 仍只做装配。

---

## Task 1: `resolveBaseURL` 纯函数

新增计算 base URL 的纯函数并以表驱动测试覆盖（含未指定地址替换、主机名保留、非法地址报错、真实随机端口绑定）。

**Files:**
- Modify: `cmd/ccfixer/root.go`
- Test: `cmd/ccfixer/root_test.go`

- [ ] **Step 1: Write the failing tests**

First update the import block at the top of `cmd/ccfixer/root_test.go` to add `net` and `strconv`:

```go
import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)
```

Then append the following to the END of `cmd/ccfixer/root_test.go`:

```go
func TestResolveBaseURL(t *testing.T) {
	cases := []struct {
		name       string
		listenAddr string
		port       int
		want       string
	}{
		{"empty host", ":0", 54321, "http://127.0.0.1:54321"},
		{"loopback", "127.0.0.1:0", 8080, "http://127.0.0.1:8080"},
		{"unspecified v4", "0.0.0.0:0", 9000, "http://127.0.0.1:9000"},
		{"unspecified v6", "[::]:0", 9100, "http://127.0.0.1:9100"},
		{"hostname", "localhost:0", 7000, "http://localhost:7000"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveBaseURL(tc.listenAddr, tc.port)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("resolveBaseURL(%q, %d) = %q, want %q", tc.listenAddr, tc.port, got, tc.want)
			}
		})
	}
}

func TestResolveBaseURLInvalid(t *testing.T) {
	if _, err := resolveBaseURL("bogus", 1234); err == nil {
		t.Fatal("want error for invalid listen address, got nil")
	}
}

func TestResolveBaseURLWithRealRandomPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	if port <= 0 {
		t.Fatalf("expected a positive bound port, got %d", port)
	}
	got, err := resolveBaseURL("127.0.0.1:0", port)
	if err != nil {
		t.Fatal(err)
	}
	want := "http://127.0.0.1:" + strconv.Itoa(port)
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./cmd/ccfixer/`
Expected: FAIL — `undefined: resolveBaseURL` (compile error).

- [ ] **Step 3: Write the implementation**

First update the import block at the top of `cmd/ccfixer/root.go` to add `net` and `strconv`:

```go
import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moonfruit/gotools/internal/ccfixer"
)
```

Then add this function to `cmd/ccfixer/root.go` (place it immediately after `runRoot`, before `newProxy`):

```go
// resolveBaseURL builds the base URL clients should use to reach the proxy,
// given the listen address the user requested and the actual bound port. An
// empty or unspecified host (e.g. ":0", "0.0.0.0", "::") is reported as
// 127.0.0.1 so the URL is directly usable.
func resolveBaseURL(listenAddr string, port int) (string, error) {
	host, _, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return "", fmt.Errorf("invalid listen address %q: %w", listenAddr, err)
	}
	if host == "" {
		host = "127.0.0.1"
	} else if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port)), nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./cmd/ccfixer/`
Expected: PASS (the 3 new tests plus the existing proxy tests). Build will succeed because `net` and `strconv` are now used by `resolveBaseURL`.

- [ ] **Step 5: Commit**

```bash
git add cmd/ccfixer/root.go cmd/ccfixer/root_test.go
git commit -m "feat(ccfixer): add resolveBaseURL helper for the listen URL"
```

---

## Task 2: 改写 `runRoot` 绑定与输出

把 `runRoot` 改为显式 `net.Listen` + `Serve`，绑定后 URL→stdout、banner→stderr；先绑定再打印（绑定失败直接返回错误）。新增一个错误路径回归测试。

**Files:**
- Modify: `cmd/ccfixer/root.go`
- Test: `cmd/ccfixer/root_test.go`

- [ ] **Step 1: Write the regression test for the bind-error path**

Append to the END of `cmd/ccfixer/root_test.go`:

```go
func TestRunRootInvalidListen(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"-u", "https://example.com", "-l", "bogus"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err == nil {
		t.Fatal("want error for invalid listen address, got nil")
	}
}
```

(This guards the error path: an unbindable listen address must return an error rather than start a server. It exercises `runRoot` without ever entering the blocking `Serve` call.)

- [ ] **Step 2: Run the test to verify it passes on current code, then keep it as a regression guard**

Run: `go test ./cmd/ccfixer/ -run TestRunRootInvalidListen -v`
Expected: PASS. (The current `ListenAndServe` path already errors on `"bogus"`; this test pins that behavior so the Task 2 refactor cannot regress it.)

- [ ] **Step 3: Rewrite `runRoot`**

In `cmd/ccfixer/root.go`, replace the entire `runRoot` function:

```go
func runRoot(cmd *cobra.Command, opts *options) error {
	target, err := url.Parse(opts.upstream)
	if err != nil {
		return fmt.Errorf("invalid upstream URL: %w", err)
	}
	if target.Scheme == "" || target.Host == "" {
		return fmt.Errorf("upstream URL must include scheme and host, got %q", opts.upstream)
	}

	proxy := newProxy(target, opts, cmd.ErrOrStderr())
	fmt.Fprintf(cmd.OutOrStdout(), "ccfixer listening on %s, forwarding to %s\n", opts.listen, opts.upstream)

	server := &http.Server{Addr: opts.listen, Handler: proxy}
	return server.ListenAndServe()
}
```

with:

```go
func runRoot(cmd *cobra.Command, opts *options) error {
	target, err := url.Parse(opts.upstream)
	if err != nil {
		return fmt.Errorf("invalid upstream URL: %w", err)
	}
	if target.Scheme == "" || target.Host == "" {
		return fmt.Errorf("upstream URL must include scheme and host, got %q", opts.upstream)
	}

	// Bind before announcing, so a bind failure (port in use, bad address)
	// returns an error instead of printing a misleading banner. A port of 0
	// in opts.listen makes the OS pick a free port, which we read back here.
	ln, err := net.Listen("tcp", opts.listen)
	if err != nil {
		return fmt.Errorf("listen on %q: %w", opts.listen, err)
	}
	baseURL, err := resolveBaseURL(opts.listen, ln.Addr().(*net.TCPAddr).Port)
	if err != nil {
		ln.Close()
		return err
	}

	proxy := newProxy(target, opts, cmd.ErrOrStderr())
	fmt.Fprintln(cmd.OutOrStdout(), baseURL)                                                                  // machine-readable
	fmt.Fprintf(cmd.ErrOrStderr(), "ccfixer listening on %s, forwarding to %s\n", ln.Addr(), opts.upstream) // human banner

	server := &http.Server{Handler: proxy}
	return server.Serve(ln)
}
```

- [ ] **Step 4: Verify build and full package tests**

Run: `go build ./... && go vet ./... && go test ./cmd/ccfixer/`
Expected: build OK; vet no output; all tests PASS (including `TestRunRootInvalidListen` and the existing proxy/`resolveBaseURL` tests). The stdout/stderr split and random-port binding are validated end-to-end by the smoke test in Task 3.

- [ ] **Step 5: Commit**

```bash
git add cmd/ccfixer/root.go cmd/ccfixer/root_test.go
git commit -m "feat(ccfixer): bind explicitly, support -l :0 random port, print URL to stdout"
```

---

## Task 3: 文档与端到端校验

更新 README 与 CLAUDE.md，跑全量校验，并用真实随机端口冒烟验证 stdout URL + 转发链路。

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`

- [ ] **Step 1: Update the README ccfixer section**

In `README.md`, find the existing ccfixer usage block:

```bash
ccfixer -u https://relay.example.com           # listen on 127.0.0.1:8787
ccfixer -u https://relay.example.com -l :9000  # custom listen address
ccfixer -u https://relay.example.com -v        # log merges per request
```

Replace that entire fenced block with:

````
```bash
ccfixer -u https://relay.example.com           # listen on 127.0.0.1:8787
ccfixer -u https://relay.example.com -l :9000  # custom listen address
ccfixer -u https://relay.example.com -l :0     # listen on a random free port
ccfixer -u https://relay.example.com -v        # log merges per request
```

The chosen base URL (`http://127.0.0.1:<port>`) is printed to **stdout** as a
single line; the human-readable banner goes to **stderr**. With `-l :0` you can
let the OS pick a port and capture the URL for shell integration:

```bash
exec 3< <(ccfixer -u https://relay.example.com -l :0)
read -r ANTHROPIC_BASE_URL <&3      # e.g. http://127.0.0.1:54321
export ANTHROPIC_BASE_URL
# ccfixer keeps serving in the background; the shell exit cleans it up
```
````

(The existing sentence right after — "Then point Claude Code at the proxy (e.g. `ANTHROPIC_BASE_URL=http://127.0.0.1:8787`)." — stays as-is.)

IMPORTANT: open and read `README.md` first to confirm the exact surrounding text before editing.

- [ ] **Step 2: Update the CLAUDE.md tool row**

In `CLAUDE.md`, the 现有工具 table has a `ccfixer` row. Replace its 说明 cell so the row reads exactly:

```markdown
| `ccfixer` | `cmd/ccfixer/` | `internal/ccfixer/` | 透明反向代理：把 Claude Code 请求 `messages` 中的 `role:"system"` 消息并入相邻 user 消息（`<system-reminder>` 包裹）后转发，响应原样透传（含流式）。`-u` 上游、`-l` 监听（`-l :0` 随机空闲端口，选定 URL 打到 stdout、banner 打到 stderr）、`-v` 详细日志。 |
```

IMPORTANT: open and read `CLAUDE.md` first to confirm the exact current row before replacing it.

- [ ] **Step 3: Full validation**

Run:
```bash
gofmt -l cmd/ccfixer
go vet ./...
go test ./...
go build ./...
```
Expected: `gofmt -l` no output; `go vet` no output; `go test ./...` all PASS; `go build` succeeds.

- [ ] **Step 4: Random-port smoke test (end-to-end)**

Run:
```bash
go build -o bin/ccfixer ./cmd/ccfixer
exec 3< <(./bin/ccfixer -u https://example.com -l :0)
read -r URL <&3
echo "captured URL: $URL"
case "$URL" in
  http://127.0.0.1:*) echo "URL format OK" ;;
  *) echo "UNEXPECTED URL FORMAT: $URL" ;;
esac
curl -s -X POST "$URL/v1/messages" \
  -H 'content-type: application/json' \
  -d '{"messages":[{"role":"user","content":"hi"},{"role":"system","content":"x"}]}' \
  -o /dev/null -w 'HTTP %{http_code}\n'
exec 3<&-   # close fd 3; the background ccfixer process exits when its stdout pipe closes
```
Expected: `captured URL: http://127.0.0.1:<some-port>` with `URL format OK`; curl prints an `HTTP <code>` line (upstream is example.com, so the exact code — likely 405 — is irrelevant; we only confirm the random-port proxy bound, printed its URL to stdout, and forwarded without crashing). `bin/` is gitignored.

- [ ] **Step 5: Commit (docs only)**

```bash
git add README.md CLAUDE.md
git commit -m "docs(ccfixer): document -l :0 random port and stdout URL"
```

---

## Self-Review 记录

- **Spec 覆盖**：触发方式（`-l :0`）→ Task 2 `runRoot` + Task 1 `resolveBaseURL`；返回方式（URL→stdout、banner→stderr）→ Task 2；URL host 推导（空/未指定→127.0.0.1，主机名保留）→ Task 1 `resolveBaseURL` + 表驱动测试；先绑定再打印 → Task 2；测试（表驱动 + 真实绑定 + 回归）→ Task 1/2；文档与 shell 惯用法 → Task 3。全部有对应任务。
- **类型一致性**：`resolveBaseURL(listenAddr string, port int) (string, error)` 在 Task 1 定义、Task 2 调用，签名一致；`runRoot(cmd *cobra.Command, opts *options) error` 签名不变。
- **占位符**：无 TODO/TBD；每个代码步骤含完整代码或完整命令。
- **已知取舍**：Task 2 的 `runRoot` 成功路径会阻塞 `Serve`，无法用普通单测直接断言其 stdout/stderr 输出；改由 Task 2 的 build+vet、`TestRunRootInvalidListen` 错误路径回归测，以及 Task 3 的随机端口冒烟测试端到端验证。已在计划中明确说明，非占位符遗漏。
