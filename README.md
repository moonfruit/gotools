# gotools

A monorepo of small Go command-line utilities. Each tool lives in `cmd/<name>/` and is independent of the others.

## Build

```bash
go build ./...                              # build everything
go build -o bin/<tool> ./cmd/<tool>         # build a single tool
```

## Tools

### `uhsort` — sort `user@host[:port]` lists

Sort lines of the form `user@host[:port][ rest]`. Sort key precedence:

1. **host** — IPv4 (numeric) < IPv6 (full-expanded) < domain (reversed-segment lex).
2. **port** — `0` means "no port", ordered before any positive port.
3. **user** — lex (case-insensitive).
4. **rest** — lex (the text after the host on the same line).

Lines that don't contain `@` are treated as host-only. Reads stdin and writes
stdout by default.

```bash
uhsort [file]            # sort
uhsort -u [file]         # dedupe equivalent lines
uhsort -c [file]         # dedupe + count prefix (TAB-separated; implies -u)
uhsort -o OUT [file]     # write to file
uhsort -i FILE           # edit FILE in place (atomic rename, mode preserved)
```

Examples:

```console
$ printf 'bob@example.com\nalice@1.1.1.1\nadmin@[::1]:22\n' | uhsort
alice@1.1.1.1
admin@[::1]:22
bob@example.com

$ printf 'a@h.com:80\nb@h.com\na@h.com:22\n' | uhsort
b@h.com
a@h.com:22
a@h.com:80

$ printf 'a@h\na@h\nb@h\n' | uhsort -c
2	a@h
1	b@h
```

Shell completion is provided by Cobra:

```bash
uhsort completion bash    # also: zsh, fish, powershell
```

### `wcwidth` — terminal display width of UTF-8 text

Compute the terminal display width of UTF-8 text: ASCII (half-width) counts as
1, CJK and full-width characters as 2, and combining marks as 0.

With positional arguments, the width of each argument is printed on its own
line. With no arguments, stdin is read and the width of each line is printed (a
trailing newline is not counted; an empty line yields 0).

East Asian Ambiguous-width characters follow the runtime environment by
default; use `-E` to force width 2 or `-N` to force width 1.

```bash
wcwidth [string ...]     # width of each argument
wcwidth                  # width of each stdin line
wcwidth -E [string ...]  # ambiguous-width characters as width 2
wcwidth -N [string ...]  # ambiguous-width characters as width 1
```

Examples:

```console
$ wcwidth hello 你好 'a😀b'
5
4
4

$ printf '中文\nascii\n' | wcwidth
4
5
```

Shell completion is provided by Cobra:

```bash
wcwidth completion bash    # also: zsh, fish, powershell
```

## License

[MIT](./LICENSE)
