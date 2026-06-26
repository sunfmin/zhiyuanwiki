package core

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
)

// MajorScoreRow 是专业录取分数线里的一行（一所院校某专业某年某科类的录取数据）。
type MajorScoreRow struct {
	Year       int
	Track      string // 科类：黑龙江=物理/历史；浙江=综合
	SchoolCode string // 院校代码
	SchoolName string // 院校名称
	GroupCode  string // 专业组代码（黑龙江逐年变；浙江多为空）
	MajorName  string // 专业名称
	SelKe      string // 选科要求
	MinScore   int    // 最低分
	MinRank    int    // 最低位次
	MaxScore   int    // 最高分
}

// NormalizeMajorName 归一化专业名：去首尾与全角空格。专业名是 (院校,专业) 叶子的稳定键，
// 不用逐年变化的专业代码。
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
