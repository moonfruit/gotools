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
// original bytes are returned untouched (no decode/re-encode round-trip), so an
// unmodified request reaches the upstream exactly as Claude Code sent it.
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

// mergeSystemMessages folds every `role:"system"` message into an adjacent user
// message, wrapping the system text in <system-reminder>...</system-reminder>.
// Preference order per system message:
//  1. the immediately preceding user message (already in the result),
//  2. else the next following user message,
//  3. else (no adjacent user) change its own role to "user".
//
// Returns the new slice and the number of system messages handled.
func mergeSystemMessages(msgs []any) ([]any, int) {
	result := make([]any, 0, len(msgs))
	count := 0
	for i := range len(msgs) {
		m, ok := msgs[i].(map[string]any)
		if !ok || m["role"] != "system" {
			result = append(result, msgs[i])
			continue
		}
		reminder := "<system-reminder>" + systemText(m) + "</system-reminder>"

		// 1. Preceding user (last element currently in result).
		if k := len(result); k > 0 {
			if prev, ok := result[k-1].(map[string]any); ok && prev["role"] == "user" {
				mergeReminder(prev, reminder)
				count++
				continue
			}
		}
		// 2. Following user.
		if j := nextUserIndex(msgs, i+1); j >= 0 {
			mergeReminder(msgs[j].(map[string]any), reminder)
			count++
			continue // drop this system message; the user at j is appended later
		}
		// 3. Fallback: relabel as user.
		m["role"] = "user"
		result = append(result, m)
		count++
	}
	return result, count
}

// systemText extracts the plain text of a system message, supporting both a
// string content and an array of content blocks (text blocks joined by "\n").
func systemText(m map[string]any) string {
	switch c := m["content"].(type) {
	case string:
		return c
	case []any:
		var parts []string
		for _, b := range c {
			blk, ok := b.(map[string]any)
			if !ok || blk["type"] != "text" {
				continue
			}
			if t, ok := blk["text"].(string); ok {
				parts = append(parts, t)
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

// mergeReminder appends the wrapped reminder to a user message's content,
// handling both string and block-array content shapes.
func mergeReminder(user map[string]any, reminder string) {
	switch c := user["content"].(type) {
	case string:
		user["content"] = c + "\n" + reminder
	case []any:
		user["content"] = append(c, map[string]any{"type": "text", "text": reminder})
	default:
		user["content"] = reminder
	}
}

// nextUserIndex returns the index of the first user message at or after `from`,
// or -1 if none.
func nextUserIndex(msgs []any, from int) int {
	for j := from; j < len(msgs); j++ {
		if m, ok := msgs[j].(map[string]any); ok && m["role"] == "user" {
			return j
		}
	}
	return -1
}
