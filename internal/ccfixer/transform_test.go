package ccfixer

import (
	"bytes"
	"testing"
)

func TestTransformPassthrough(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"invalid json", `not json at all`},
		{"no messages field", `{"model":"m","max_tokens":1024}`},
		{"messages not array", `{"messages":"oops"}`},
		{"no system message", `{"messages":[{"role":"user","content":"hi"},{"role":"assistant","content":"yo"}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := []byte(tc.input)
			out, n, err := Transform(in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if n != 0 {
				t.Fatalf("want n=0, got %d", n)
			}
			if !bytes.Equal(out, in) {
				t.Fatalf("want output unchanged\n in: %s\nout: %s", in, out)
			}
		})
	}
}
