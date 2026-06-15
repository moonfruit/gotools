package ccfixer

import (
	"bytes"
	"encoding/json"
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

func decodeBody(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, b)
	}
	return m
}

func messagesOf(t *testing.T, m map[string]any) []any {
	t.Helper()
	msgs, ok := m["messages"].([]any)
	if !ok {
		t.Fatalf("messages missing or not array: %v", m["messages"])
	}
	return msgs
}

func msgAt(t *testing.T, msgs []any, i int) map[string]any {
	t.Helper()
	mm, ok := msgs[i].(map[string]any)
	if !ok {
		t.Fatalf("message %d is not an object: %v", i, msgs[i])
	}
	return mm
}

func TestMergeSystemAfterUserString(t *testing.T) {
	in := []byte(`{"messages":[
		{"role":"user","content":"hello"},
		{"role":"system","content":"be terse"},
		{"role":"assistant","content":"hi"}
	]}`)
	out, n, err := Transform(in)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want n=1, got %d", n)
	}
	msgs := messagesOf(t, decodeBody(t, out))
	if len(msgs) != 2 {
		t.Fatalf("want 2 messages, got %d", len(msgs))
	}
	u := msgAt(t, msgs, 0)
	if u["role"] != "user" {
		t.Fatalf("message 0 role = %v, want user", u["role"])
	}
	want := "hello\n<system-reminder>be terse</system-reminder>"
	if u["content"] != want {
		t.Fatalf("content = %q, want %q", u["content"], want)
	}
	if msgAt(t, msgs, 1)["role"] != "assistant" {
		t.Fatalf("message 1 should be assistant")
	}
}

func TestMergeSystemAfterUserBlocks(t *testing.T) {
	in := []byte(`{"messages":[
		{"role":"user","content":[{"type":"text","text":"hello"}]},
		{"role":"system","content":[{"type":"text","text":"be terse"}]}
	]}`)
	out, n, err := Transform(in)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want n=1, got %d", n)
	}
	msgs := messagesOf(t, decodeBody(t, out))
	if len(msgs) != 1 {
		t.Fatalf("want 1 message, got %d", len(msgs))
	}
	blocks, ok := msgAt(t, msgs, 0)["content"].([]any)
	if !ok || len(blocks) != 2 {
		t.Fatalf("want content array of 2 blocks, got %v", msgAt(t, msgs, 0)["content"])
	}
	last, _ := blocks[1].(map[string]any)
	if last["type"] != "text" || last["text"] != "<system-reminder>be terse</system-reminder>" {
		t.Fatalf("last block = %v", last)
	}
}

func TestMergeSystemAfterAssistantUsesFollowingUser(t *testing.T) {
	in := []byte(`{"messages":[
		{"role":"user","content":"a"},
		{"role":"assistant","content":"b"},
		{"role":"system","content":"ctx"},
		{"role":"user","content":"c"}
	]}`)
	out, n, err := Transform(in)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want n=1, got %d", n)
	}
	msgs := messagesOf(t, decodeBody(t, out))
	if len(msgs) != 3 {
		t.Fatalf("want 3 messages, got %d", len(msgs))
	}
	if msgAt(t, msgs, 1)["role"] != "assistant" {
		t.Fatalf("message 1 should be assistant")
	}
	u := msgAt(t, msgs, 2)
	want := "c\n<system-reminder>ctx</system-reminder>"
	if u["role"] != "user" || u["content"] != want {
		t.Fatalf("message 2 = %v, want user content %q", u, want)
	}
}

func TestMergeMultipleSystem(t *testing.T) {
	in := []byte(`{"messages":[
		{"role":"user","content":"u1"},
		{"role":"system","content":"s1"},
		{"role":"system","content":"s2"},
		{"role":"assistant","content":"a1"}
	]}`)
	out, n, err := Transform(in)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("want n=2, got %d", n)
	}
	msgs := messagesOf(t, decodeBody(t, out))
	if len(msgs) != 2 {
		t.Fatalf("want 2 messages, got %d", len(msgs))
	}
	want := "u1\n<system-reminder>s1</system-reminder>\n<system-reminder>s2</system-reminder>"
	if msgAt(t, msgs, 0)["content"] != want {
		t.Fatalf("content = %q, want %q", msgAt(t, msgs, 0)["content"], want)
	}
}

func TestMergeSystemNoUserFallsBackToRoleChange(t *testing.T) {
	in := []byte(`{"messages":[{"role":"system","content":"x"}]}`)
	out, n, err := Transform(in)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want n=1, got %d", n)
	}
	msgs := messagesOf(t, decodeBody(t, out))
	if len(msgs) != 1 {
		t.Fatalf("want 1 message, got %d", len(msgs))
	}
	if msgAt(t, msgs, 0)["role"] != "user" {
		t.Fatalf("fallback should change role to user, got %v", msgAt(t, msgs, 0)["role"])
	}
}

func TestTransformDoesNotHTMLEscapeAndKeepsNumbers(t *testing.T) {
	in := []byte(`{"max_tokens":1024,"messages":[{"role":"user","content":"hi"},{"role":"system","content":"x"}]}`)
	out, _, err := Transform(in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte("<system-reminder>x</system-reminder>")) {
		t.Fatalf("system-reminder should be literal, got: %s", out)
	}
	if !bytes.Contains(out, []byte("1024")) || bytes.Contains(out, []byte("1024.0")) {
		t.Fatalf("max_tokens number formatting changed: %s", out)
	}
}
