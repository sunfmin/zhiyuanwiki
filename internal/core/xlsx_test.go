package core

import "testing"

func TestNormSchoolCode(t *testing.T) {
	tests := []struct{ in, want string }{
		{"1", "0001"},
		{"31", "0031"},
		{"123", "0123"},
		{"4620", "4620"},
		{" 1 ", "0001"},
		{"", ""},
		{"12345", "12345"}, // ≥4 位不截断
		{"A1", "A1"},       // 非纯数字原样
	}
	for _, tt := range tests {
		if got := NormSchoolCode(tt.in); got != tt.want {
			t.Errorf("NormSchoolCode(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseLeadingInt(t *testing.T) {
	tests := []struct {
		in   string
		want int
		ok   bool
	}{
		{"700以上", 700, true},
		{"693-750", 693, true},
		{"  280  ", 280, true},
		{"待定", 0, false},
		{"", 0, false},
	}
	for _, tt := range tests {
		got, ok := ParseLeadingInt(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("ParseLeadingInt(%q) = (%d,%v), want (%d,%v)", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}
