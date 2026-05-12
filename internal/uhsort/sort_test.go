package uhsort

import (
	"reflect"
	"testing"
)

func parseAll(ss []string) []Line {
	out := make([]Line, len(ss))
	for i, s := range ss {
		out[i] = Parse(s)
	}
	return out
}

func raws(lines []Line) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.Raw
	}
	return out
}

func TestSort_HostThenPortThenUserThenRest(t *testing.T) {
	in := parseAll([]string{
		"bob@example.com",
		"alice@1.1.1.1",
		"admin@[::1]:22",
		"alice@example.com:80",
		"alice@example.com:22",
		"bob@example.com:22",
		"alice@example.com:22 z",
		"alice@example.com:22 a",
	})
	Sort(in)
	want := []string{
		"alice@1.1.1.1",
		"admin@[::1]:22",
		"bob@example.com",
		"alice@example.com:22",
		"alice@example.com:22 a",
		"alice@example.com:22 z",
		"bob@example.com:22",
		"alice@example.com:80",
	}
	if got := raws(in); !reflect.DeepEqual(got, want) {
		t.Errorf("Sort order mismatch\n got = %v\nwant = %v", got, want)
	}
}

func TestSort_Stable(t *testing.T) {
	// Two fully equal lines should retain input order via stable sort.
	in := parseAll([]string{
		"alice@example.com first",
		"alice@example.com first",
	})
	in[0].Raw = "A"
	in[1].Raw = "B"
	Sort(in)
	if in[0].Raw != "A" || in[1].Raw != "B" {
		t.Errorf("stable sort should keep A before B, got %v", raws(in))
	}
}

func TestDedupe(t *testing.T) {
	in := parseAll([]string{
		"alice@example.com",
		"alice@example.com",
		"bob@example.com",
		"alice@example.com",
		"alice@Example.COM", // case-insensitive host equivalence
	})
	out := Dedupe(in)
	got := raws(out)
	want := []string{"alice@example.com", "bob@example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Dedupe mismatch\n got = %v\nwant = %v", got, want)
	}
}

func TestDedupe_RestSensitive(t *testing.T) {
	// Different Rest → not deduped.
	in := parseAll([]string{
		"a@h.com  x",
		"a@h.com  y",
		"a@h.com  x",
	})
	out := Dedupe(in)
	if len(out) != 2 {
		t.Errorf("expected 2 unique lines, got %d: %v", len(out), raws(out))
	}
}

func TestCount(t *testing.T) {
	in := parseAll([]string{
		"a@h",
		"a@h",
		"b@h",
		"a@h",
	})
	cs := Count(in)
	if len(cs) != 2 {
		t.Fatalf("expected 2 groups, got %d: %+v", len(cs), cs)
	}
	if cs[0].N != 3 || cs[0].Line.Raw != "a@h" {
		t.Errorf("group 0: got (%d,%q) want (3,%q)", cs[0].N, cs[0].Line.Raw, "a@h")
	}
	if cs[1].N != 1 || cs[1].Line.Raw != "b@h" {
		t.Errorf("group 1: got (%d,%q) want (1,%q)", cs[1].N, cs[1].Line.Raw, "b@h")
	}
}
