package wcwidth

import "testing"

func TestWidth(t *testing.T) {
	yes, no := true, false
	tests := []struct {
		name      string
		in        string
		eastAsian *bool
		want      int
	}{
		{"empty", "", nil, 0},
		{"ascii", "abc", nil, 3},
		{"cjk", "你好", nil, 4},
		{"mixed", "你好world", nil, 9},
		{"combining acute decomposed", "é", nil, 1},
		{"ambiguous forced wide", "±", &yes, 2},
		{"ambiguous forced narrow", "±", &no, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Width(tt.in, tt.eastAsian); got != tt.want {
				t.Errorf("Width(%q, %v) = %d, want %d", tt.in, tt.eastAsian, got, tt.want)
			}
		})
	}
}
