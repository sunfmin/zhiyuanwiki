package hlj

import "testing"

// 取自官方 2026 物理类一分一段表顶部的真实形状（含"700以上"顶段）。
func fixtureTable() *YiFenYiDuan {
	rows := [][]string{
		{"黑龙江省2026年普通高考物理类文化课一分段", "", ""},
		{"", "", ""},
		{"分段", "分段人数", "累计人数"},
		{"700以上", "17", "17"},
		{"699", "7", "24"},
		{"698", "6", "30"},
		{"697", "7", "37"},
		{"696", "3", "40"},
	}
	y, err := parseYiFenYiDuanRows(rows, "黑龙江", "物理", 2026)
	if err != nil {
		panic(err)
	}
	return y
}

func TestParseYiFenYiDuanRows(t *testing.T) {
	y := fixtureTable()
	if len(y.Entries) != 5 {
		t.Fatalf("解析到 %d 条，想要 5", len(y.Entries))
	}
	// 升序排列：第一条应是 696。
	if y.Entries[0].Score != 696 || y.Entries[0].Cumulative != 40 {
		t.Errorf("Entries[0] = %+v，想要 {696, ., 40}", y.Entries[0])
	}
	// "700以上" 顶段 Score 应解析为 700。
	last := y.Entries[len(y.Entries)-1]
	if last.Score != 700 || last.Cumulative != 17 {
		t.Errorf("顶段 = %+v，想要 {700, ., 17}", last)
	}
}

func TestScoreToRank(t *testing.T) {
	y := fixtureTable()
	tests := []struct {
		name  string
		score int
		want  int
	}{
		{"精确-顶段", 700, 17},
		{"精确-699", 699, 24},
		{"精确-最低段696", 696, 40},
		{"高于顶段-取顶段累计", 720, 17},
		{"缺失分-就近向上取698", 698, 30},
		{"低于最低段-取最低段", 690, 40},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := y.ScoreToRank(tt.score)
			if !ok {
				t.Fatalf("ScoreToRank(%d) ok=false", tt.score)
			}
			if got != tt.want {
				t.Errorf("ScoreToRank(%d) = %d，想要 %d", tt.score, got, tt.want)
			}
		})
	}
}

func TestRankToScore(t *testing.T) {
	y := fixtureTable()
	tests := []struct {
		name string
		rank int
		want int
	}{
		{"顶段内rank10-返回最高分", 10, 700},
		{"rank17边界-700", 17, 700},
		{"rank18落入699段", 18, 699},
		{"rank24边界-699", 24, 699},
		{"rank25落入698段", 25, 698},
		{"超过表底-返回最低分", 9999, 696},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := y.RankToScore(tt.rank)
			if !ok {
				t.Fatalf("RankToScore(%d) ok=false", tt.rank)
			}
			if got != tt.want {
				t.Errorf("RankToScore(%d) = %d，想要 %d", tt.rank, got, tt.want)
			}
		})
	}
}

func TestScoreToRankEmpty(t *testing.T) {
	y := &YiFenYiDuan{}
	if _, ok := y.ScoreToRank(600); ok {
		t.Error("空表 ScoreToRank 应 ok=false")
	}
}
