package uhsort

import (
	"bytes"
	"net"
	"net/netip"
	"strconv"
	"strings"
)

// Category 是 host 的大类，决定主排序顺序：IPv4 < IPv6 < Domain。
type Category int

const (
	CatIPv4 Category = iota
	CatIPv6
	CatDomain
)

// Key 是用于比较的归一化 host 排序键。
//
//   - IPv4: Bytes[:4] 为 4 字节大端 IP；
//   - IPv6: Bytes[:16] 为完整展开的 16 字节；
//   - Domain: Dom 为反向分段（先 TLD，后子域），全部 lower。
//
// Port 始终参与比较，0 表示"无 port"，按数值序排在所有正 port 之前。
type Key struct {
	Cat   Category
	Bytes [16]byte
	Dom   []string
	Port  uint16
}

// SplitHostPort 把 host token 拆为 host 字符串和可选 port。
// 识别顺序：
//  1. "ip:port" / "[v6]:port" → netip.ParseAddrPort
//  2. 裸 IP（含 "::1"、"2001:db8::1"）→ netip.ParseAddr
//  3. "domain:port" → net.SplitHostPort
//  4. 其余整体当作 host（域名），port 为 0
func SplitHostPort(token string) (host string, port uint16) {
	if token == "" {
		return "", 0
	}
	if ap, err := netip.ParseAddrPort(token); err == nil {
		return ap.Addr().String(), ap.Port()
	}
	if a, err := netip.ParseAddr(token); err == nil {
		return a.String(), 0
	}
	if h, p, err := net.SplitHostPort(token); err == nil {
		if pn, err := strconv.ParseUint(p, 10, 16); err == nil {
			return h, uint16(pn)
		}
	}
	return token, 0
}

// Classify 根据 host 字符串与 port 计算排序键。
func Classify(host string, port uint16) Key {
	k := Key{Port: port}
	if a, err := netip.ParseAddr(host); err == nil {
		if a.Is4() {
			k.Cat = CatIPv4
			b := a.As4()
			copy(k.Bytes[:4], b[:])
			return k
		}
		k.Cat = CatIPv6
		b := a.As16()
		copy(k.Bytes[:], b[:])
		return k
	}
	k.Cat = CatDomain
	if host == "" {
		return k
	}
	lower := strings.ToLower(host)
	parts := strings.Split(lower, ".")
	reversed := make([]string, len(parts))
	for i, p := range parts {
		reversed[len(parts)-1-i] = p
	}
	k.Dom = reversed
	return k
}

// Less 实现完整的 host+port 比较。
func (a Key) Less(b Key) bool {
	if a.Cat != b.Cat {
		return a.Cat < b.Cat
	}
	switch a.Cat {
	case CatIPv4:
		if c := bytes.Compare(a.Bytes[:4], b.Bytes[:4]); c != 0 {
			return c < 0
		}
	case CatIPv6:
		if c := bytes.Compare(a.Bytes[:], b.Bytes[:]); c != 0 {
			return c < 0
		}
	case CatDomain:
		n := min(len(a.Dom), len(b.Dom))
		for i := range n {
			if a.Dom[i] != b.Dom[i] {
				return a.Dom[i] < b.Dom[i]
			}
		}
		if len(a.Dom) != len(b.Dom) {
			return len(a.Dom) < len(b.Dom)
		}
	}
	return a.Port < b.Port
}
