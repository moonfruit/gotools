package ccfixer

import (
	"strings"
	"testing"
)

// Extra scenario: two system messages both pointing to the same following user
func TestMergeMultipleSystemToSameFollowingUser(t *testing.T) {
	in := []byte(`{"messages":[
		{"role":"system","content":"s1"},
		{"role":"system","content":"s2"},
		{"role":"user","content":"hello"}
	]}`)
	out, n, err := Transform(in)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("want n=2, got %d", n)
	}
	msgs := messagesOf(t, decodeBody(t, out))
	if len(msgs) != 1 {
		t.Fatalf("want 1 message, got %d: %v", len(msgs), msgs)
	}
	content, _ := msgAt(t, msgs, 0)["content"].(string)
	if !strings.Contains(content, "<system-reminder>s1</system-reminder>") ||
		!strings.Contains(content, "<system-reminder>s2</system-reminder>") {
		t.Fatalf("both reminders should be in content, got %q", content)
	}
}

// Extra scenario: following user has block-array content
func TestMergeSystemFollowingUserBlocks(t *testing.T) {
	in := []byte(`{"messages":[
		{"role":"assistant","content":"yo"},
		{"role":"system","content":"hint"},
		{"role":"user","content":[{"type":"text","text":"q"}]}
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
	blocks, ok := msgAt(t, msgs, 1)["content"].([]interface{})
	if !ok || len(blocks) != 2 {
		t.Fatalf("want 2 content blocks, got %v", msgAt(t, msgs, 1)["content"])
	}
	last, _ := blocks[1].(map[string]interface{})
	if last["type"] != "text" || last["text"] != "<system-reminder>hint</system-reminder>" {
		t.Fatalf("last block unexpected: %v", last)
	}
}
