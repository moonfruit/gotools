package uhsort

import "testing"

func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		in       string
		wantHost string
		wantPort uint16
	}{
		{"", "", 0},
		{"1.2.3.4", "1.2.3.4", 0},
		{"1.2.3.4:22", "1.2.3.4", 22},
		{"::1", "::1", 0},
		{"2001:db8::1", "2001:db8::1", 0},
		{"[::1]:22", "::1", 22},
		{"example.com", "example.com", 0},
		{"example.com:8080", "example.com", 8080},
		{"weird-thing", "weird-thing", 0},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			h, p := SplitHostPort(tt.in)
			if h != tt.wantHost || p != tt.wantPort {
				t.Errorf("SplitHostPort(%q) = (%q,%d); want (%q,%d)", tt.in, h, p, tt.wantHost, tt.wantPort)
			}
		})
	}
}

// asserts a < b in Key.Less, and b not < a.
func assertLess(t *testing.T, name string, a, b Key) {
	t.Helper()
	if !a.Less(b) {
		t.Errorf("%s: expected a < b but Less returned false (a=%+v b=%+v)", name, a, b)
	}
	if b.Less(a) {
		t.Errorf("%s: expected !(b < a) but Less returned true (a=%+v b=%+v)", name, a, b)
	}
}

func mkKey(token string) Key {
	h, p := SplitHostPort(token)
	return Classify(h, p)
}

func TestKeyLess_Category(t *testing.T) {
	// IPv4 < IPv6 < Domain
	assertLess(t, "ipv4 < ipv6", mkKey("1.1.1.1"), mkKey("::1"))
	assertLess(t, "ipv6 < domain", mkKey("::1"), mkKey("a.com"))
	assertLess(t, "ipv4 < domain", mkKey("255.255.255.255"), mkKey("a.com"))
	// port does not cross category
	assertLess(t, "ipv4:65535 < ipv6", mkKey("1.2.3.4:65535"), mkKey("::1"))
}

func TestKeyLess_IPv4Numeric(t *testing.T) {
	// String lex would say "10..." > "192..."; numeric should put 10 first.
	assertLess(t, "10 < 192", mkKey("10.0.0.1"), mkKey("192.168.0.1"))
	assertLess(t, "1.2.3.4 < 1.2.3.5", mkKey("1.2.3.4"), mkKey("1.2.3.5"))
}

func TestKeyLess_IPv6Expanded(t *testing.T) {
	assertLess(t, "::1 < ::2", mkKey("::1"), mkKey("::2"))
	assertLess(t, "fe80::1 < fe80::2", mkKey("fe80::1"), mkKey("fe80::2"))
	// :: vs 0:0:... should be equal (both 16 zero bytes)
	a := mkKey("::")
	b := mkKey("0:0:0:0:0:0:0:0")
	if a.Less(b) || b.Less(a) {
		t.Errorf(":: and 0:0:0:0:0:0:0:0 should compare equal: a=%+v b=%+v", a, b)
	}
}

func TestKeyLess_DomainReversed(t *testing.T) {
	// reversed segments
	assertLess(t, "example.com < example.net", mkKey("example.com"), mkKey("example.net"))
	assertLess(t, "a.example.com < b.example.com", mkKey("a.example.com"), mkKey("b.example.com"))
	// short < long with same suffix
	assertLess(t, "example.com < mail.example.com", mkKey("example.com"), mkKey("mail.example.com"))
}

func TestKeyLess_Port(t *testing.T) {
	assertLess(t, "no-port < :22", mkKey("1.2.3.4"), mkKey("1.2.3.4:22"))
	assertLess(t, ":22 < :80", mkKey("1.2.3.4:22"), mkKey("1.2.3.4:80"))
	assertLess(t, "domain no-port < :443", mkKey("example.com"), mkKey("example.com:443"))
}

func TestClassify_CaseInsensitive(t *testing.T) {
	a := mkKey("Example.COM")
	b := mkKey("example.com")
	if a.Less(b) || b.Less(a) {
		t.Errorf("case-folded hosts should compare equal: a=%+v b=%+v", a, b)
	}
}
