package uhsort

import (
	"strings"
	"unicode"
)

// Line 表示一行 "user@host[:port][ rest]" 的解析结果。
// Raw 保留输入原文（不含行末换行），输出时按 Raw 写出。
type Line struct {
	User string
	Host string
	Port uint16
	Rest string
	Raw  string
}

// Parse 解析一行：
//  1. 首个 '@' 之前为 User；无 '@' 则 User 为空。
//  2. '@' 之后到首个空白前为 host token；空白及之后为 Rest（含前导空白）。
//  3. host token 交给 SplitHostPort 拆出 Host 与 Port。
func Parse(raw string) Line {
	l := Line{Raw: raw}
	tail := raw
	if i := strings.IndexByte(tail, '@'); i >= 0 {
		l.User = tail[:i]
		tail = tail[i+1:]
	}
	hostToken := tail
	for i, r := range tail {
		if unicode.IsSpace(r) {
			hostToken = tail[:i]
			l.Rest = tail[i:]
			break
		}
	}
	l.Host, l.Port = SplitHostPort(hostToken)
	return l
}
