package hlj

import "testing"

func metaRows() [][]string {
	return [][]string{
		{"年份", "学校", "省份", "城市", "_985", "_211", "双一流", "办学性质", "学校类别"},
		{"2020", "北京大学", "北京", "北京市", "是", "是", "是", "公办", "综合类"},
		{"2020", "哈尔滨理工大学", "黑龙江", "哈尔滨市", "否", "否", "否", "公办", "理工类"},
		{"2020", "中国石油大学(华东)", "山东", "青岛市", "否", "是", "是", "公办", "理工类"},
		{"2020", "某民办学院", "河北", "廊坊市", "否", "否", "否", "民办", "综合类"},
	}
}

func TestSchoolMetaLookup(t *testing.T) {
	si := NewSchoolMetaIndex()
	si.AddRows(metaRows())

	tests := []struct {
		name      string
		query     string
		wantFound bool
		want      SchoolMeta
	}{
		{"全列解析", "北京大学", true,
			SchoolMeta{Province: "北京", City: "北京市", Ownership: "公办", Kind: "综合类", Is985: true, Is211: true, IsShuangYiLiu: true}},
		{"无层次但有属性也收录", "哈尔滨理工大学", true,
			SchoolMeta{Province: "黑龙江", City: "哈尔滨市", Ownership: "公办", Kind: "理工类"}},
		{"民办", "某民办学院", true,
			SchoolMeta{Province: "河北", City: "廊坊市", Ownership: "民办", Kind: "综合类"}},
		{"分校去括号继承母体", "中国石油大学(北京)", true,
			SchoolMeta{Province: "山东", City: "青岛市", Ownership: "公办", Kind: "理工类", Is211: true, IsShuangYiLiu: true}},
		{"未知院校", "野鸡大学", false, SchoolMeta{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := si.Lookup(tt.query)
			if found != tt.wantFound {
				t.Fatalf("Lookup(%q) found = %v, want %v", tt.query, found, tt.wantFound)
			}
			if got != tt.want {
				t.Errorf("Lookup(%q) = %+v, want %+v", tt.query, got, tt.want)
			}
		})
	}
}

func TestSchoolMetaLevels(t *testing.T) {
	si := NewSchoolMetaIndex()
	si.AddRows(metaRows())
	m, _ := si.Lookup("北京大学")
	got := m.Levels()
	want := []string{"985", "211", "双一流"}
	if len(got) != len(want) {
		t.Fatalf("Levels() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Levels()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if m2, _ := si.Lookup("哈尔滨理工大学"); len(m2.Levels()) != 0 {
		t.Errorf("无层次院校 Levels() = %v, want []", m2.Levels())
	}
}

func TestSchoolMetaMergeFirstNonEmpty(t *testing.T) {
	// 跨文件合并：字符串取首个非空、布尔按位 OR。
	si := NewSchoolMetaIndex()
	si.AddRows([][]string{
		{"学校", "省份", "城市", "办学性质", "双一流"},
		{"测试大学", "江苏", "", "公办", "否"},
	})
	si.AddRows([][]string{
		{"学校", "省份", "城市", "办学性质", "双一流"},
		{"测试大学", "上海", "南京市", "", "是"}, // 省份已有不覆盖；城市补空；双一流 OR
	})
	got, _ := si.Lookup("测试大学")
	want := SchoolMeta{Province: "江苏", City: "南京市", Ownership: "公办", IsShuangYiLiu: true}
	if got != want {
		t.Errorf("merged = %+v, want %+v", got, want)
	}
}

func TestSchoolMetaNoColumns(t *testing.T) {
	// 表头无任何院校属性列（如新格式招生计划）应安全跳过。
	si := NewSchoolMetaIndex()
	si.AddRows([][]string{
		{"年份", "院校代码", "专业名称", "计划人数"},
		{"2026", "1001", "数学类", "5"},
	})
	if si.Len() != 0 {
		t.Errorf("Len() = %d, want 0", si.Len())
	}
}
