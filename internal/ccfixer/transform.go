// Package ccfixer rewrites Claude Code's Anthropic Messages API request bodies
// so that mid-conversation `role:"system"` messages are merged into an adjacent
// user message, for upstreams that reject the system role inside `messages`.
package ccfixer

import (
	"bytes"
	"encoding/json"
	"strings"
)

// Transform takes a raw request body and returns the rewritten body together
// with the number of system messages that were merged. It is fail-open: any
// problem (non-JSON, missing/!array `messages`, encode failure) yields the
// original bytes unchanged with n=0 and a nil error. When nothing is merged the
// original bytes are returned untouched, preserving byte-for-byte stability for
// upstream prefix caching.
func Transform(body []byte) (out []byte, n int, err error) {
	var root map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber() // keep numbers as json.Number so they re-encode verbatim
	if e := dec.Decode(&root); e != nil {
		return body, 0, nil
	}
	msgs, ok := root["messages"].([]any)
	if !ok {
		return body, 0, nil
	}
	newMsgs, count := mergeSystemMessages(msgs)
	if count == 0 {
		return body, 0, nil
	}
	root["messages"] = newMsgs

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // keep "<system-reminder>" literal, not <...
	if e := enc.Encode(root); e != nil {
		return body, 0, nil
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), count, nil
}

// mergeSystemMessages is fully implemented in Task 2. For now it is a no-op so
// the passthrough behavior compiles and the no-system case returns count 0.
func mergeSystemMessages(msgs []any) ([]any, int) {
	return msgs, 0
}

var _ = strings.Join // referenced by Task 2; keeps the import while stubbed
