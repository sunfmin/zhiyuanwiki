package core

import "testing"

func newTestMenlei() *MenleiClassifier {
	mc := NewMenleiClassifier()
	mc.addRows([][]string{
		{"年份", "学校", "门类", "专业"},
		{"2020", "某大学", "理学", "数学类（数学与应用数学、信息与计算科学）"},
		{"2020", "某大学", "管理学", "工商管理类（会计学、财务管理）"},
		{"2020", "某大学", "医学", "临床医学"},
		// 高职专科「大类」噪声行：门类不在 12 门类内，应被忽略。
		{"2020", "某专科", "装备制造大类", "工业机器人技术"},
	})
	return mc
}

func TestMenleiCode(t *testing.T) {
	mc := newTestMenlei()
	tests := []struct {
		name  string
		major string
		want  string
	}{
		{"精确-去括号基名命中", "数学类", "理"},
		{"精确-全名命中", "数学类（数学与应用数学、信息与计算科学）", "理"},
		{"精确-管理学", "工商管理类", "管"},
		{"精确-医学", "临床医学", "医"},
		{"大类噪声不污染映射-退到关键词", "工业机器人技术", "工"}, // 含「机器人/技术」→ 工
		{"关键词-工学", "智能制造工程", "工"},
		{"关键词-农学优先于医（动物医学）", "动物医学", "农"},
		{"关键词-文学优先于法（法语）", "法语", "文"},
		{"关键词-法学", "法学", "法"},
		{"关键词-经济学", "数字金融", "经"},
		{"关键词-教育学", "学前教育", "教"},
		{"未命中归其他", "星际航行玄学", "他"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mc.Code(tt.major); got != tt.want {
				t.Errorf("Code(%q) = %q, want %q", tt.major, got, tt.want)
			}
		})
	}
}

func TestMenleiAddRowsNoColumns(t *testing.T) {
	mc := NewMenleiClassifier()
	mc.addRows([][]string{
		{"年份", "院校代码", "专业名称", "计划人数"}, // 无门类列
		{"2026", "1001", "数学类", "5"},
	})
	if mc.Len() != 0 {
		t.Errorf("Len() = %d, want 0（无门类列应跳过）", mc.Len())
	}
}
