package hlj

import "testing"

func tagRows() [][]string {
	return [][]string{
		{"年份", "学校", "省份", "_985", "_211", "双一流", "办学性质"},
		{"2023", "北京大学", "北京", "是", "是", "是", "公办"},
		{"2023", "郑州大学", "河南", "否", "是", "是", "公办"},
		{"2023", "宁波大学", "浙江", "否", "否", "是", "公办"},
		{"2023", "哈尔滨理工大学", "黑龙江", "否", "否", "否", "公办"},
		{"2023", "中国石油大学(华东)", "山东", "否", "是", "是", "公办"},
	}
}

func TestTagIndexLookup(t *testing.T) {
	ti := NewTagIndex()
	ti.AddRows(tagRows())

	tests := []struct {
		name      string
		query     string
		wantFound bool
		want      SchoolTag
	}{
		{"985 全标签", "北京大学", true, SchoolTag{Is985: true, Is211: true, IsShuangYiLiu: true}},
		{"211 非985", "郑州大学", true, SchoolTag{Is211: true, IsShuangYiLiu: true}},
		{"仅双一流", "宁波大学", true, SchoolTag{IsShuangYiLiu: true}},
		{"无标签院校不收录", "哈尔滨理工大学", false, SchoolTag{}},
		{"全角括号归一化", "中国石油大学（华东）", true, SchoolTag{Is211: true, IsShuangYiLiu: true}},
		{"分校去括号继承母体", "中国石油大学(北京)", true, SchoolTag{Is211: true, IsShuangYiLiu: true}},
		{"未知院校", "野鸡大学", false, SchoolTag{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, found := ti.Lookup(tc.query)
			if found != tc.wantFound {
				t.Fatalf("Lookup(%q) found = %v, want %v", tc.query, found, tc.wantFound)
			}
			if got != tc.want {
				t.Errorf("Lookup(%q) = %+v, want %+v", tc.query, got, tc.want)
			}
		})
	}
}

func TestTagIndexMergeOR(t *testing.T) {
	// 同一院校跨文件出现，标签按位 OR 合并（某年缺标记不应抹掉另一年的）。
	ti := NewTagIndex()
	ti.AddRows([][]string{
		{"学校", "_985", "_211", "双一流"},
		{"测试大学", "否", "是", "否"},
	})
	ti.AddRows([][]string{
		{"学校", "_985", "_211", "双一流"},
		{"测试大学", "否", "否", "是"},
	})
	got, found := ti.Lookup("测试大学")
	if !found {
		t.Fatal("merged school not found")
	}
	want := SchoolTag{Is211: true, IsShuangYiLiu: true}
	if got != want {
		t.Errorf("merged = %+v, want %+v", got, want)
	}
}

func TestTagIndexNoTagColumns(t *testing.T) {
	// 表头无层次列（如新格式招生计划）时应安全跳过，不收录任何院校。
	ti := NewTagIndex()
	ti.AddRows([][]string{
		{"年份", "院校代码", "院校名称", "专业名称", "计划人数"},
		{"2026", "1001", "北京大学", "数学类", "5"},
	})
	if ti.Len() != 0 {
		t.Errorf("Len() = %d, want 0 (no tag columns)", ti.Len())
	}
}
