package hlj

import "testing"

func TestNormalizeMajorName(t *testing.T) {
	tests := []struct{ in, want string }{
		{" 计算机科学与技术 ", "计算机科学与技术"},
		{"金融　学", "金融学"},
		{"数据 科学", "数据科学"},
	}
	for _, tt := range tests {
		if got := NormalizeMajorName(tt.in); got != tt.want {
			t.Errorf("NormalizeMajorName(%q) = %q，想要 %q", tt.in, got, tt.want)
		}
	}
}

func TestMajorKey(t *testing.T) {
	// 确定性 + 对归一化前的空格不敏感。
	if MajorKey("计算机科学与技术") != MajorKey(" 计算机科学与技术 ") {
		t.Error("MajorKey 应对首尾空格不敏感")
	}
	if MajorKey("计算机科学与技术") == MajorKey("软件工程") {
		t.Error("不同专业名应得不同 key")
	}
	if len(MajorKey("法学")) != 8 {
		t.Errorf("MajorKey 应为 8 位十六进制，得 %q", MajorKey("法学"))
	}
}

func TestParseMajorScoreRows(t *testing.T) {
	rows := [][]string{
		{"黑龙江2025年专业分数线", "", "", "", "", "", "", "", "", "", "", "", "", ""},
		{"年份", "生源地", "批次", "科类", "院校代码", "院校名称", "专业组代码", "专业代码", "专业名称", "专业备注", "选科要求", "最低分", "最低位次", "最高分"},
		{"2025", "黑龙江", "本科批", "物理", "1003", "清华大学", "012", "57", "计算机科学与技术", "", "化学", "690", "120", "700"},
		{"2025", "黑龙江", "本科批", "历史", "1666", "安徽财经大学", "001", "06", "日语", "", "不限", "463", "12988", "515"},
		// 旧科类（理科）应被排除
		{"2025", "黑龙江", "本科批", "理科", "9999", "某校", "001", "01", "机械", "", "不限", "400", "50000", "450"},
		// 缺最低位次应被排除
		{"2025", "黑龙江", "本科批", "物理", "8888", "无位次校", "001", "01", "测试", "", "不限", "400", "", "450"},
		// 提前批（非本科批）应被排除
		{"2025", "黑龙江", "提前批", "物理", "7777", "提前校", "001", "01", "测试", "", "不限", "400", "100", "450"},
	}
	got, err := parseMajorScoreRows(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("解析到 %d 行，想要 2（物理+历史本科批含位次）", len(got))
	}
	if got[0].SchoolName != "清华大学" || got[0].MinRank != 120 || got[0].Track != "物理" {
		t.Errorf("第一行 = %+v", got[0])
	}
}

func TestAggregateLeaves(t *testing.T) {
	rows := []MajorScoreRow{
		{Year: 2024, Track: "物理", SchoolCode: "1003", SchoolName: "清华大学", MajorName: "计算机科学与技术", SelKe: "化学", MinScore: 688, MinRank: 150, MaxScore: 700},
		{Year: 2025, Track: "物理", SchoolCode: "1003", SchoolName: "清华大学", MajorName: "计算机科学与技术", SelKe: "化学", MinScore: 690, MinRank: 120, MaxScore: 700},
		// 同年同科类同叶子的第二条，位次更低（更难）应胜出
		{Year: 2025, Track: "物理", SchoolCode: "1003", SchoolName: "清华大学", MajorName: "计算机科学与技术", SelKe: "化学", MinScore: 692, MinRank: 100, MaxScore: 701},
		{Year: 2025, Track: "物理", SchoolCode: "1003", SchoolName: "清华大学", MajorName: "法学", SelKe: "不限", MinScore: 680, MinRank: 400, MaxScore: 690},
	}
	schools, leaves := AggregateLeaves(rows)
	if len(schools) != 1 || schools[0].Code != "1003" {
		t.Fatalf("schools = %+v", schools)
	}
	if len(leaves) != 2 {
		t.Fatalf("leaves = %d，想要 2（计算机 + 法学）", len(leaves))
	}
	// 找计算机叶子
	var cs *MajorLeaf
	for i := range leaves {
		if leaves[i].MajorName == "计算机科学与技术" {
			cs = &leaves[i]
		}
	}
	if cs == nil {
		t.Fatal("缺计算机叶子")
	}
	if len(cs.Years) != 2 {
		t.Fatalf("计算机应有 2 年走势，得 %d", len(cs.Years))
	}
	// 2025 应取最低位次 100（最难那条）
	y2025 := cs.Years[len(cs.Years)-1]
	if y2025.Year != 2025 || y2025.MinRank != 100 {
		t.Errorf("2025 数据点 = %+v，想要 {2025,...,100,...}", y2025)
	}
}
