package uhsort

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want Line
	}{
		{
			name: "user@host",
			in:   "alice@host.com",
			want: Line{User: "alice", Host: "host.com", Port: 0, Rest: "", Raw: "alice@host.com"},
		},
		{
			name: "user@host:port",
			in:   "alice@host.com:22",
			want: Line{User: "alice", Host: "host.com", Port: 22, Rest: "", Raw: "alice@host.com:22"},
		},
		{
			name: "host only",
			in:   "host.com",
			want: Line{User: "", Host: "host.com", Port: 0, Rest: "", Raw: "host.com"},
		},
		{
			name: "with rest preserving leading whitespace",
			in:   "alice@host.com  # prod",
			want: Line{User: "alice", Host: "host.com", Port: 0, Rest: "  # prod", Raw: "alice@host.com  # prod"},
		},
		{
			name: "ipv4 with port",
			in:   "u@1.2.3.4:22",
			want: Line{User: "u", Host: "1.2.3.4", Port: 22, Rest: "", Raw: "u@1.2.3.4:22"},
		},
		{
			name: "bracketed ipv6 with port",
			in:   "u@[::1]:22",
			want: Line{User: "u", Host: "::1", Port: 22, Rest: "", Raw: "u@[::1]:22"},
		},
		{
			name: "bare ipv6 no port",
			in:   "u@2001:db8::1",
			want: Line{User: "u", Host: "2001:db8::1", Port: 0, Rest: "", Raw: "u@2001:db8::1"},
		},
		{
			name: "multiple at signs splits at first",
			in:   "a@b@host.com",
			want: Line{User: "a", Host: "b@host.com", Port: 0, Rest: "", Raw: "a@b@host.com"},
		},
		{
			name: "empty",
			in:   "",
			want: Line{Raw: ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.in)
			if got != tt.want {
				t.Errorf("Parse(%q):\n got = %+v\nwant = %+v", tt.in, got, tt.want)
			}
		})
	}
}
