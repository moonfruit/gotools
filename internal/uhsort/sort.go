package uhsort

import (
	"slices"
	"strings"
)

// CountedLine 是 Count 结果中带计数的行。
type CountedLine struct {
	N    int
	Line Line
}

type sortKey struct {
	host Key
	user string
	rest string
}

func keyFor(l Line) sortKey {
	return sortKey{
		host: Classify(l.Host, l.Port),
		user: strings.ToLower(l.User),
		rest: l.Rest,
	}
}

func cmpKey(a, b sortKey) int {
	if a.host.Less(b.host) {
		return -1
	}
	if b.host.Less(a.host) {
		return 1
	}
	if a.user != b.user {
		if a.user < b.user {
			return -1
		}
		return 1
	}
	if a.rest != b.rest {
		if a.rest < b.rest {
			return -1
		}
		return 1
	}
	return 0
}

// Sort 原地稳定排序 lines，排序键为 (host, port, user, rest)。
func Sort(lines []Line) {
	type bundled struct {
		line Line
		key  sortKey
	}
	bs := make([]bundled, len(lines))
	for i, l := range lines {
		bs[i] = bundled{l, keyFor(l)}
	}
	slices.SortStableFunc(bs, func(a, b bundled) int { return cmpKey(a.key, b.key) })
	for i, b := range bs {
		lines[i] = b.line
	}
}

// equalForDedupe 决定两行在去重时是否视为等价。
func equalForDedupe(a, b Line) bool {
	return strings.EqualFold(a.User, b.User) &&
		strings.EqualFold(a.Host, b.Host) &&
		a.Port == b.Port &&
		a.Rest == b.Rest
}

// Dedupe 先排序再原地去重；返回去重后的子切片，保留首次出现行的 Raw。
func Dedupe(lines []Line) []Line {
	Sort(lines)
	out := lines[:0]
	for _, l := range lines {
		if len(out) > 0 && equalForDedupe(out[len(out)-1], l) {
			continue
		}
		out = append(out, l)
	}
	return out
}

// Count 先排序再聚合等价行，返回每组的代表行与出现次数。
func Count(lines []Line) []CountedLine {
	Sort(lines)
	var out []CountedLine
	for _, l := range lines {
		if n := len(out); n > 0 && equalForDedupe(out[n-1].Line, l) {
			out[n-1].N++
			continue
		}
		out = append(out, CountedLine{N: 1, Line: l})
	}
	return out
}
