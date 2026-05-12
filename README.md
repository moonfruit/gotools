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

## License

[MIT](./LICENSE)
