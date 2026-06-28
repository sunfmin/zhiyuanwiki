package core

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
)

// YiFenYiDuan 是某省某科类某年的一分一段表：分数 → 累计位次。
// Entries 按分数升序排列；累计人数即该分数的位次。
type YiFenYiDuan struct {
	Province    string         `json:"province"`
	Track       string         `json:"track"`
	Year        int            `json:"year"`
	ControlLine int            `json:"controlLine,omitempty"` // 本科批控制线（特控线），源一分一段表自带；缺则 0/省略
	Entries     []FenduanEntry `json:"entries"`
}

// FenduanEntry 一个分数段：Score 该分数，Count 该分数人数，
// Cumulative 累计人数（= 该分数的位次，即全省得分 ≥ Score 的人数）。
type FenduanEntry struct {
	Score      int `json:"score"`
	Count      int `json:"count"`
	Cumulative int `json:"cumulative"`
}

// ScoreToRank 把分数换算成位次：取"分数 ≥ score 的最小分数段"的累计人数。
// 累计人数 = 得分 ≥ 该分数的人数，正是位次的定义。此规则同时正确处理：
//   - "X以上"顶段（如 700以上 存为 Score=700，覆盖 ≥700）
//   - 缺失分（就近向上取最接近的已列分数段）
//   - 高于最高段的分数（返回顶段累计，作为最好上界）
func (y *YiFenYiDuan) ScoreToRank(score int) (rank int, ok bool) {
	if len(y.Entries) == 0 {
		return 0, false
	}
	// Entries 升序。找第一个 Score >= score 的段。
	i := sort.Search(len(y.Entries), func(i int) bool { return y.Entries[i].Score >= score })
	if i == len(y.Entries) {
		// score 高于最高段：用最高分段累计作为最好上界。
		return y.Entries[len(y.Entries)-1].Cumulative, true
	}
	return y.Entries[i].Cumulative, true
}

// RankToScore 把位次换算成分数：取"累计人数 ≥ rank 的最高分数段"。
// 累计随分数下降而增大，故位次为 rank 的考生分数 = 满足 累计≥rank 的最高分。
func (y *YiFenYiDuan) RankToScore(rank int) (score int, ok bool) {
	if len(y.Entries) == 0 || rank < 1 {
		return 0, false
	}
	// Entries 升序 by Score（=> Cumulative 降序）。从最高分往低扫，
	// 第一个 Cumulative >= rank 的分数即所求最高分。
	for i := len(y.Entries) - 1; i >= 0; i-- {
		if y.Entries[i].Cumulative >= rank {
			return y.Entries[i].Score, true
		}
	}
	// rank 超过最低段累计（落在表底之外）：返回最低分作为下界。
	return y.Entries[0].Score, true
}

// Total 返回该科类该年的考生总人数（=最低分段的累计人数）。
func (y *YiFenYiDuan) Total() int {
	if len(y.Entries) == 0 {
		return 0
	}
	return y.Entries[0].Cumulative // Entries 升序，最低分累计最大
}

// SortFenduanAscending 把分数段按分数升序排稳（Entries 升序是 ScoreToRank/Total 的前置不变量）。
func SortFenduanAscending(y *YiFenYiDuan) {
	sort.Slice(y.Entries, func(i, j int) bool { return y.Entries[i].Score < y.Entries[j].Score })
}

// ParseYiFenYiDuanXLSX 解析官方一分一段 xlsx（表头驱动，容忍标题行/空行/列序不同）。
// 列识别：累计列含"累计"，人数列含"人数"，分数列含"分段"或"分数"。
func ParseYiFenYiDuanXLSX(path, province, track string, year int) (*YiFenYiDuan, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("打开 %s: %w", path, err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("%s: 无 sheet", path)
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("读 %s: %w", path, err)
	}
	return parseYiFenYiDuanRows(rows, province, track, year)
}

// parseYiFenYiDuanRows 是纯逻辑部分，便于用内存中的行表做单元测试。
func parseYiFenYiDuanRows(rows [][]string, province, track string, year int) (*YiFenYiDuan, error) {
	headerIdx := -1
	for i, r := range rows {
		for _, c := range r {
			if strings.Contains(c, "累计") {
				headerIdx = i
				break
			}
		}
		if headerIdx >= 0 {
			break
		}
	}
	if headerIdx < 0 {
		return nil, fmt.Errorf("未找到含\"累计\"的表头行")
	}

	header := rows[headerIdx]
	scoreCol, countCol, cumCol := 0, 1, len(header)-1
	for i, c := range header {
		switch {
		case strings.Contains(c, "累计"):
			cumCol = i
		case strings.Contains(c, "人数"):
			countCol = i
		case strings.Contains(c, "分段") || strings.Contains(c, "分数"):
			scoreCol = i
		}
	}

	y := &YiFenYiDuan{Province: province, Track: track, Year: year}
	for _, r := range rows[headerIdx+1:] {
		score, ok := ParseLeadingInt(Cell(r, scoreCol))
		if !ok {
			continue
		}
		cum, ok := ParseLeadingInt(Cell(r, cumCol))
		if !ok {
			continue
		}
		count, _ := ParseLeadingInt(Cell(r, countCol))
		y.Entries = append(y.Entries, FenduanEntry{Score: score, Count: count, Cumulative: cum})
	}
	if len(y.Entries) == 0 {
		return nil, fmt.Errorf("表头之后未解析到数据行")
	}
	sort.Slice(y.Entries, func(i, j int) bool { return y.Entries[i].Score < y.Entries[j].Score })
	return y, nil
}
