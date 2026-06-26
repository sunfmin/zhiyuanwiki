// Package zj 是浙江高考数据的省份专属解析：专业录取分数线（综合·一段/二段/提前批）、
// 2026 招生计划→院校×专业报考视图、一表联动院校/专业属性。浙江是单科类「综合」+
// 专业平行志愿（填报单位是院校×专业，无组内调剂），与黑龙江结构不同。
// 省份无关的聚合/换算/键/门类等用 internal/core。
package zj

import (
	"fmt"
	"strings"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/xuri/excelize/v2"
)

// Track 是浙江唯一科类。
const Track = "综合"

// batchKeep 判定一个批次是否属于本站收录范围：普通类一段/二段/提前批/平行录取
// （命名逐年变：旧年份「平行录取一段/二段」，新年份「普通类一段/二段」）。
// 艺术/体育已由「科类=综合」过滤掉；专项计划批不含这些关键词，自然排除。
func batchKeep(batch string) bool {
	return strings.Contains(batch, "一段") ||
		strings.Contains(batch, "二段") ||
		strings.Contains(batch, "提前批") ||
		strings.Contains(batch, "平行录取")
}

// ParseMajorScoresXLSX 解析浙江「全国高校在浙江的专业录取分数」xlsx，只返回科类=综合、
// 收录批次、且含最低位次的行。表头驱动；浙江列名：专业/最低分数/最低位次（无最高分）。
func ParseMajorScoresXLSX(path string) ([]core.MajorScoreRow, error) {
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
	return parseMajorScoreRows(rows)
}

func parseMajorScoreRows(rows [][]string) ([]core.MajorScoreRow, error) {
	headerIdx := -1
	for i, r := range rows {
		if core.HasCell(r, "院校代码") && core.HasCell(r, "最低位次") {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return nil, fmt.Errorf("未找到含\"院校代码\"+\"最低位次\"的表头行")
	}
	h := rows[headerIdx]
	col := func(names ...string) int { return core.FindCol(h, names...) }
	cYear := col("年份")
	cTrack := col("科类")
	cBatch := col("批次", "批次名称")
	cSchoolCode := col("院校代码")
	cSchoolName := col("院校名称")
	cMajor := col("专业", "专业名称")
	cSelKe := col("选科要求")
	cMin := col("最低分数", "最低分")
	cRank := col("最低位次")

	var out []core.MajorScoreRow
	for _, r := range rows[headerIdx+1:] {
		if strings.TrimSpace(core.Cell(r, cTrack)) != Track {
			continue
		}
		if !batchKeep(core.Cell(r, cBatch)) {
			continue
		}
		minRank, hasRank := core.ParseLeadingInt(core.Cell(r, cRank))
		if !hasRank {
			continue // 位次缺失行不进位次模型
		}
		name := strings.TrimSpace(core.Cell(r, cMajor))
		code := core.NormSchoolCode(core.Cell(r, cSchoolCode))
		if name == "" || code == "" {
			continue
		}
		year, _ := core.ParseLeadingInt(core.Cell(r, cYear))
		minScore, _ := core.ParseLeadingInt(core.Cell(r, cMin))
		out = append(out, core.MajorScoreRow{
			Year:       year,
			Track:      Track,
			SchoolCode: code,
			SchoolName: strings.TrimSpace(core.Cell(r, cSchoolName)),
			MajorName:  name,
			SelKe:      strings.TrimSpace(core.Cell(r, cSelKe)),
			MinScore:   minScore,
			MinRank:    minRank,
			MaxScore:   0, // 浙江源表无最高分
		})
	}
	return out, nil
}
