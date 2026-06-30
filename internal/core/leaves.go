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

// StripParenTail 截去专业名从首个全/半角括号起的尾注（校区/「包含专业：…」/办学地点/语种等）。
// 部分省的招生计划用带尾注的专业名，而录取分数表用裸名；截断后两表才能按 (院校,专业名) 挂接。
// 录取分数表本就无括号（各省均验证为 0），故对其调用是恒等；仅在招生计划解析处按需使用。
func StripParenTail(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "（("); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
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

// rankOrInf 把「无位次」(≤0) 视作正无穷，便于「取最低位次」比较里让有位次的行总胜出。
func rankOrInf(rank int) int {
	if rank <= 0 {
		return int(^uint(0) >> 1) // math.MaxInt
	}
	return rank
}

// maxInt 返回若干整数里的最大值（用于无位次省聚合分数跨度上界）。
func maxInt(xs ...int) int {
	m := xs[0]
	for _, x := range xs[1:] {
		if x > m {
			m = x
		}
	}
	return m
}

// AggregateLeaves 把行表聚合成院校列表与院校×专业叶子。
// 叶子键 = 院校代码 + 归一化专业名；同一叶子同年同科类多行取最低位次（最难那条）。
// 无位次省（西藏「只有分数」）退化为取最低分代表、并记分数跨度上界于 MaxScore。
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
		cand := YearScore{Year: r.Year, Track: r.Track, MinScore: r.MinScore, MinRank: r.MinRank, MaxScore: r.MaxScore}
		switch {
		case cur == nil:
			leafYears[leafID][yk] = &ypoint{ys: cand}
		case cur.ys.MinRank > 0 || r.MinRank > 0:
			// 有位次省：维持原行为——同年同科类取最低位次（最难）那条。无位次记 +∞、不会胜出。
			if rankOrInf(r.MinRank) < rankOrInf(cur.ys.MinRank) {
				leafYears[leafID][yk] = &ypoint{ys: cand}
			}
		default:
			// 都无位次（西藏「只有分数」，且 A 类/B 类两线未区分常致同键多分数）：代表取最低分（最易达
			// 的线即入），并把该 年×科类 的分数跨度上界记进 MaxScore，供叶子页展示「录取分 最低–最高」。
			lo := cur.ys.MinScore
			if r.MinScore < lo {
				lo = r.MinScore
			}
			hi := maxInt(cur.ys.MinScore, cur.ys.MaxScore, r.MinScore, r.MaxScore)
			cur.ys = YearScore{Year: r.Year, Track: r.Track, MinScore: lo, MinRank: 0, MaxScore: hi}
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
