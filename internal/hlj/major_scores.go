package hlj

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
)

// MajorScoreRow 是专业录取分数线里的一行（一所院校某专业某年的录取数据）。
type MajorScoreRow struct {
	Year       int
	Track      string // 科类：物理 / 历史
	SchoolCode string // 院校代码
	SchoolName string // 院校名称
	GroupCode  string // 专业组代码（逐年变）
	MajorName  string // 专业名称
	SelKe      string // 选科要求
	MinScore   int    // 最低分
	MinRank    int    // 最低位次
	MaxScore   int    // 最高分
}

// 新科类口径（物理/历史）。旧高考的理科/文科不进位次模型。
var newGaokaoTracks = map[string]bool{"物理": true, "历史": true}

// NormalizeMajorName 归一化专业名：去首尾与全角空格。专业名是 (院校,专业) 叶子的稳定键，
// 不用逐年变化的专业代码。更精细的专业类归并见 slice ⑥。
func NormalizeMajorName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "　", "") // 全角空格
	s = strings.ReplaceAll(s, " ", "")
	return s
}

// MajorKey 由归一化专业名生成确定性短哈希，作为叶子页 URL 段（ascii、跨重建稳定）。
func MajorKey(majorName string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(NormalizeMajorName(majorName)))
	return fmt.Sprintf("%08x", h.Sum32())
}

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

func hasCell(row []string, s string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) == s {
			return true
		}
	}
	return false
}

// findCol 返回表头中精确等于任一候选名的列下标；找不到返回 -1。
func findCol(header []string, names ...string) int {
	for i, c := range header {
		cc := strings.TrimSpace(c)
		for _, n := range names {
			if cc == n {
				return i
			}
		}
	}
	return -1
}

// ── 聚合：把多年行表整理成院校与院校×专业叶子 ──────────────────

// School 是稳定主干实体。
type School struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// YearScore 院校×专业某年某科类的录取数据点。
type YearScore struct {
	Year     int    `json:"year"`
	Track    string `json:"track"`
	MinScore int    `json:"minScore"`
	MinRank  int    `json:"minRank"`
	MaxScore int    `json:"maxScore"`
}

// MajorLeaf 是数据叶子：某院校的某专业（按院校+专业名稳定），含历年走势。
type MajorLeaf struct {
	SchoolCode string      `json:"schoolCode"`
	MajorKey   string      `json:"majorKey"`
	MajorName  string      `json:"majorName"`
	SelKe      string      `json:"selKe"` // 最近年份的选科要求
	Years      []YearScore `json:"years"` // 按 (年份,科类) 升序

	yearSeen int // 内部：用于取最新年份选科要求，不序列化
}

// AggregateLeaves 把行表聚合成院校列表与院校×专业叶子。
// 叶子键 = 院校代码 + 归一化专业名；同一叶子同年同科类多行取最低位次（最难那条）。
func AggregateLeaves(rows []MajorScoreRow) ([]School, []MajorLeaf) {
	schoolName := map[string]string{}
	schoolYear := map[string]int{} // 记录用以取最新校名

	type ypoint struct {
		ys      YearScore
		selKe   string
		selYear int
	}
	leafYears := map[string]map[string]*ypoint{} // leafID -> "year|track" -> point
	leafMeta := map[string]*MajorLeaf{}

	for _, r := range rows {
		if r.SchoolCode == "" {
			continue
		}
		if r.Year >= schoolYear[r.SchoolCode] {
			schoolYear[r.SchoolCode] = r.Year
			schoolName[r.SchoolCode] = r.SchoolName
		}
		key := MajorKey(r.MajorName)
		leafID := r.SchoolCode + "/" + key
		if leafMeta[leafID] == nil {
			leafMeta[leafID] = &MajorLeaf{
				SchoolCode: r.SchoolCode,
				MajorKey:   key,
				MajorName:  NormalizeMajorName(r.MajorName),
			}
			leafYears[leafID] = map[string]*ypoint{}
		}
		// 最新年份的选科要求作为叶子选科
		if r.Year >= leafMeta[leafID].yearSeen {
			leafMeta[leafID].yearSeen = r.Year
			leafMeta[leafID].SelKe = r.SelKe
		}
		yk := fmt.Sprintf("%d|%s", r.Year, r.Track)
		cur := leafYears[leafID][yk]
		if cur == nil || r.MinRank < cur.ys.MinRank {
			leafYears[leafID][yk] = &ypoint{ys: YearScore{
				Year: r.Year, Track: r.Track,
				MinScore: r.MinScore, MinRank: r.MinRank, MaxScore: r.MaxScore,
			}}
		}
	}

	schools := make([]School, 0, len(schoolName))
	for code, name := range schoolName {
		schools = append(schools, School{Code: code, Name: name})
	}
	sort.Slice(schools, func(i, j int) bool { return schools[i].Code < schools[j].Code })

	leaves := make([]MajorLeaf, 0, len(leafMeta))
	for id, m := range leafMeta {
		for _, p := range leafYears[id] {
			m.Years = append(m.Years, p.ys)
		}
		sort.Slice(m.Years, func(i, j int) bool {
			if m.Years[i].Year != m.Years[j].Year {
				return m.Years[i].Year < m.Years[j].Year
			}
			return m.Years[i].Track < m.Years[j].Track
		})
		leaves = append(leaves, *m)
	}
	sort.Slice(leaves, func(i, j int) bool {
		if leaves[i].SchoolCode != leaves[j].SchoolCode {
			return leaves[i].SchoolCode < leaves[j].SchoolCode
		}
		return leaves[i].MajorName < leaves[j].MajorName
	})
	return schools, leaves
}
