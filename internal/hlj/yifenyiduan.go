package hlj

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

// ParseYiFenYiDuan 解析黑龙江一分一段 xlsx → 各科类（物理/历史）一张表。黑龙江源是异构的，
// 本函数据表头自适配两种形态：
//
//   - 逐科类表（2024 及万师兄 2026）：表头仅「分数 人数 累计人数」，科类在文件名里
//     （物理类/历史类）。旧高考的理科/文科表无法映射到物理/历史，站点也不用，返回 nil 跳过。
//   - 合表（2025「黑龙江202X年的一分一段表」）：单表多科类，列含 科类/批次/控制线/分数/累计。
//     本科批 + 专科批两段拼成全分布（专科段补足本科线以下的低分），控制线取本科批那一行。
//
// 与 internal/js 的合表解析同形，只是黑龙江多了「逐科类文件 + 本/专科两段」的形态差异。
func ParseYiFenYiDuan(path, province string, year int) ([]*core.YiFenYiDuan, error) {
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

	if isCombinedYFD(rows) {
		return parseCombinedYFD(rows, province, year)
	}
	track := trackFromName(filepath.Base(path))
	if track == "" {
		return nil, nil // 旧高考理科/文科逐科类表：站点只用物理/历史，跳过（非错误）
	}
	y, err := core.ParseYiFenYiDuanRows(rows, province, track, year)
	if err != nil {
		return nil, err
	}
	return []*core.YiFenYiDuan{y}, nil
}

// isCombinedYFD 判定是否为「单表多科类」合表：靠前若干行里有「科类」列。
func isCombinedYFD(rows [][]string) bool {
	for i := 0; i < len(rows) && i < 5; i++ {
		if core.HasCell(rows[i], "科类") {
			return true
		}
	}
	return false
}

// parseCombinedYFD 解析合表：按科类分组（物理/历史），本科批 + 专科批拼成全分布，
// 控制线取本科批行（同年同科类各行相同）。
func parseCombinedYFD(rows [][]string, province string, year int) ([]*core.YiFenYiDuan, error) {
	s, err := core.NewSheet(rows, func(r []string) bool {
		return core.HasCell(r, "科类") && core.HasCellContains(r, "累计")
	})
	if err != nil {
		return nil, err
	}
	cTrack, cBatch := s.Col("科类"), s.Col("批次")
	cControl := s.ColContains("控制线")
	cScore := s.ColContains("分数", "分段")
	cCount := s.ColContains("本段人数", "段人数")
	cCum := s.ColContains("累计")

	byTrack := map[string]*core.YiFenYiDuan{}
	var order []string
	for _, r := range s.Data {
		track := canonTrack(core.Cell(r, cTrack))
		if track != "物理" && track != "历史" {
			continue // 旧高考理科/文科或空行
		}
		score, ok := core.ParseLeadingInt(core.Cell(r, cScore))
		if !ok {
			continue
		}
		cum, ok := core.ParseLeadingInt(core.Cell(r, cCum))
		if !ok {
			continue
		}
		count, _ := core.ParseLeadingInt(core.Cell(r, cCount))
		y := byTrack[track]
		if y == nil {
			y = &core.YiFenYiDuan{Province: province, Track: track, Year: year}
			byTrack[track] = y
			order = append(order, track)
		}
		// 控制线只取本科批（专科批控制线是专科线，非本科线），同年同科类各行相同，取首个。
		if y.ControlLine == 0 && strings.Contains(core.Cell(r, cBatch), "本科批") {
			if cl, ok := core.ParseLeadingInt(core.Cell(r, cControl)); ok {
				y.ControlLine = cl
			}
		}
		y.Entries = append(y.Entries, core.FenduanEntry{Score: score, Count: count, Cumulative: cum})
	}
	out := make([]*core.YiFenYiDuan, 0, len(order))
	for _, t := range order {
		core.SortFenduanAscending(byTrack[t])
		out = append(out, byTrack[t])
	}
	return out, nil
}

// trackFromName 从逐科类文件名识别科类（物理/历史）；旧高考理科/文科等返回空串。
func trackFromName(name string) string {
	switch {
	case strings.Contains(name, "物理"):
		return "物理"
	case strings.Contains(name, "历史"):
		return "历史"
	}
	return ""
}

// canonTrack 把「物理类/历史类」归一成站点科类名（物理/历史）；其余原样返回（由调用方过滤）。
func canonTrack(s string) string {
	switch strings.TrimSpace(s) {
	case "物理类":
		return "物理"
	case "历史类":
		return "历史"
	}
	return strings.TrimSpace(s)
}
