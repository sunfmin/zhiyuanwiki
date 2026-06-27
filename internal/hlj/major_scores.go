package hlj

import (
	"strings"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

// 新科类口径（物理/历史）。旧高考的理科/文科不进位次模型。
var newGaokaoTracks = map[string]bool{"物理": true, "历史": true}

// majorScoreHeader 判定专业录取分数线表的表头行（含院校代码 + 专业名称）。
func majorScoreHeader(r []string) bool {
	return core.HasCell(r, "院校代码") && core.HasCell(r, "专业名称")
}

// ParseMajorScoresXLSX 解析一个年份的专业录取分数线 xlsx，只返回新科类（物理/历史）、
// 本科批、且含最低位次的行。表头驱动（容忍有/无标题行、列序不同、sheet 名不同）。
func ParseMajorScoresXLSX(path string) ([]core.MajorScoreRow, error) {
	s, err := core.OpenSheet(path, majorScoreHeader)
	if err != nil {
		return nil, err
	}
	return parseMajorScoreSheet(s), nil
}

func parseMajorScoreSheet(s *core.Sheet) []core.MajorScoreRow {
	col := s.Col
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

	var out []core.MajorScoreRow
	for _, r := range s.Data {
		track := strings.TrimSpace(core.Cell(r, cTrack))
		if !newGaokaoTracks[track] {
			continue
		}
		if !strings.Contains(core.Cell(r, cBatch), "本科") {
			continue
		}
		year, _ := core.ParseLeadingInt(core.Cell(r, col("年份")))
		minRank, hasRank := core.ParseLeadingInt(core.Cell(r, cRank))
		if !hasRank {
			continue // 位次缺失行不进位次模型
		}
		minScore, _ := core.ParseLeadingInt(core.Cell(r, cMin))
		maxScore, _ := core.ParseLeadingInt(core.Cell(r, cMax))
		name := strings.TrimSpace(core.Cell(r, cMajor))
		if name == "" {
			continue
		}
		out = append(out, core.MajorScoreRow{
			Year:       year,
			Track:      track,
			SchoolCode: strings.TrimSpace(core.Cell(r, cSchoolCode)),
			SchoolName: strings.TrimSpace(core.Cell(r, cSchoolName)),
			GroupCode:  strings.TrimSpace(core.Cell(r, cGroup)),
			MajorName:  name,
			SelKe:      strings.TrimSpace(core.Cell(r, cSelKe)),
			MinScore:   minScore,
			MinRank:    minRank,
			MaxScore:   maxScore,
		})
	}
	return out
}
