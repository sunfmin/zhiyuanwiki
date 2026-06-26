package hlj

import "testing"

func TestIsCoop(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  bool
	}{
		{"专业名含中外", []string{"计算机科学与技术(中外合作)", "", ""}, true},
		{"备注含合作办学", []string{"软件工程", "软件工程", "中外合作办学项目"}, true},
		{"全称含中外", []string{"工商管理", "工商管理(中外合作办学)", ""}, true},
		{"普通专业", []string{"临床医学", "临床医学", ""}, false},
		{"空", []string{"", "", ""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCoop(tt.parts...); got != tt.want {
				t.Errorf("IsCoop(%v) = %v, want %v", tt.parts, got, tt.want)
			}
		})
	}
}

func TestParseTuition(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"5000", 5000},
		{"26000", 26000},
		{"待定", 0},
		{"", 0},
		{"学费待定", 0},
		{"20000元/年", 20000},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := ParseTuition(tt.in); got != tt.want {
				t.Errorf("ParseTuition(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
