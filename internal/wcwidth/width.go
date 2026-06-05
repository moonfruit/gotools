// Package wcwidth computes the terminal display width of UTF-8 text.
package wcwidth

import "github.com/mattn/go-runewidth"

// Width returns the display width of s.
//
// Half-width characters (such as ASCII) count as 1, CJK and full-width
// characters as 2, and combining marks as 0.
//
// eastAsian controls how East Asian Ambiguous-width characters are measured.
// When nil, they follow the runtime environment ($LANG/$LC_*). When non-nil,
// *eastAsian forces ambiguous characters to width 2 (true) or width 1 (false).
func Width(s string, eastAsian *bool) int {
	cond := runewidth.DefaultCondition
	if eastAsian != nil {
		cond = &runewidth.Condition{EastAsianWidth: *eastAsian}
	}
	return cond.StringWidth(s)
}
