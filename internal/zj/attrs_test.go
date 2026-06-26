package zj

import "testing"

func TestCityTierOf(t *testing.T) {
	tests := []struct{ in, want string }{
		{"新一线城市/省会城市", "新一线"},
		{"一线城市", "一线"},
		{"二线城市/省会城市", "二线"},
		{"三线城市", "三线"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := cityTierOf(tt.in); got != tt.want {
			t.Errorf("cityTierOf(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTagsOf(t *testing.T) {
	is985, is211, syl := tagsOf("985/211/双一流/国重点/保研资格")
	if !is985 || !is211 || !syl {
		t.Errorf("解析 985/211/双一流 失败：%v %v %v", is985, is211, syl)
	}
	is985, is211, syl = tagsOf("省重点/示范高职/双高计划")
	if is985 || is211 || syl {
		t.Errorf("非顶尖院校不应命中：%v %v %v", is985, is211, syl)
	}
	is985, is211, syl = tagsOf("211/双一流/保研资格")
	if is985 || !is211 || !syl {
		t.Errorf("211/双一流（非985）：%v %v %v", is985, is211, syl)
	}
}

func TestAttrIndexMerge(t *testing.T) {
	ai := NewAttrIndex()
	// 一表联动：城市/层级/类型/双一流
	ai.addLianRows([][]string{
		{"2025招生计划"}, // 超表头
		{"院校代码", "院校名称", "所在省", "城市", "城市水平标签", "院校标签", "类型", "公私性质"},
		{"0001", "浙江大学", "浙江", "杭州", "新一线城市/省会城市", "985/211/双一流/保研资格", "综合", "公办"},
	})
	// 专业录取分数：兜底省/性质/985/211（这里补一个一表联动没有的院校）
	ai.addScoreRows([][]string{
		{"年份", "院校名称", "院校代码", "科类", "批次", "专业", "选科要求", "最低分数", "最低位次", "学校所在", "学校性质", "是否985", "是否211"},
		{"2025", "浙江大学", "0001", "综合", "普通类一段", "x", "不限", "660", "1200", "浙江", "公办", "是", "是"},
		{"2025", "某独立学院", "0009", "综合", "普通类二段", "y", "不限", "480", "180000", "海南", "民办", "否", "否"},
	})

	a, ok := ai.Lookup("0001")
	if !ok {
		t.Fatal("缺 0001")
	}
	if a.Province != "浙江" || a.City != "杭州" || a.CityTier != "新一线" || a.Kind != "综合" {
		t.Errorf("0001 = %+v", a)
	}
	if !a.Is985 || !a.Is211 || !a.IsShuangYiLiu {
		t.Errorf("0001 层次 = %v", a.Levels())
	}
	// 仅出现在分数表的院校也应有省/性质
	b, ok := ai.Lookup("0009")
	if !ok || b.Province != "海南" || b.Ownership != "民办" {
		t.Errorf("0009 = %+v ok=%v", b, ok)
	}
}
