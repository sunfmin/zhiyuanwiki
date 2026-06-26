package hlj

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// 新科类口径（物理/历史）。旧高考的理科/文科不进位次模型。
var newGaokaoTracks = map[string]bool{"物理": true, "历史": true}

// ParseMajorScoresXLSX 解析一个年份的专业录取分数线 xlsx，只返回新科类（物理/历史）、
// 本科批、且含最低位次的行。表头驱动（容忍有/无标题行、列序不同、sheet 名不同）。
func ParseMajorScoresXLSX(path string) ([]MajorScoreRow, error) {
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

func parseMajorScoreRows(rows [][]string) ([]MajorScoreRow, error) {
	headerIdx := -1
	for i, r := range rows {
		if hasCell(r, "院校代码") && hasCell(r, "专业名称") {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return nil, fmt.Errorf("未找到含\"院校代码\"+\"专业名称\"的表头行")
	}
	h := rows[headerIdx]
	col := func(names ...string) int { return findCol(h, names...) }
	cTrack := col("科类")
	cBatch := col("批次", "批次名称")
	cSchoolCode := col("院校代码")
	cSchoolName := col("院校名称")
	cGroup := col("专业组代码")
	cMajor := col("专业名称")
	cSelKe := col("选科要求")
	cMin := col("最低分")
	cRank := col("最低位次")
	cMax := col("最高分")

	var out []MajorScoreRow
	for _, r := range rows[headerIdx+1:] {
		track := strings.TrimSpace(cell(r, cTrack))
		if !newGaokaoTracks[track] {
			continue
		}
		if !strings.Contains(cell(r, cBatch), "本科") {
			continue
		}
		year, _ := parseLeadingInt(cell(r, col("年份")))
		minRank, hasRank := parseLeadingInt(cell(r, cRank))
		if !hasRank {
			continue // 位次缺失行不进位次模型
		}
		minScore, _ := parseLeadingInt(cell(r, cMin))
		maxScore, _ := parseLeadingInt(cell(r, cMax))
		name := strings.TrimSpace(cell(r, cMajor))
		if name == "" {
			continue
		}
		out = append(out, MajorScoreRow{
			Year:       year,
			Track:      track,
			SchoolCode: strings.TrimSpace(cell(r, cSchoolCode)),
			SchoolName: strings.TrimSpace(cell(r, cSchoolName)),
			GroupCode:  strings.TrimSpace(cell(r, cGroup)),
			MajorName:  name,
			SelKe:      strings.TrimSpace(cell(r, cSelKe)),
			MinScore:   minScore,
			MinRank:    minRank,
			MaxScore:   maxScore,
		})
	}
	return out, nil
}
