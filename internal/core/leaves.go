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

// School 是稳定主干实体。Key（归一化校名）是主键与 URL slug（ADR-0021）；Code 是代表代号，
// 仅供展示「院校代码」，非主键——多渠道校（普通/中外/专项）会共用一个 Key、只留主渠道代表代号。
type School struct {
	Key  string `json:"key"`
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

// MajorLeaf 是数据叶子：某院校的某专业（按 院校实体键 + 渠道 + 专业名 稳定），含历年走势。
// SchoolKey 是所属院校实体键（归一化校名，聚到院校页）；SchoolCode 是所属招生渠道的代表代号
// （区分普通/专项/中外，避免同名专业不同渠道的录取线相混）。
type MajorLeaf struct {
	SchoolKey  string      `json:"schoolKey"`
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

// AggregateLeaves 把行表聚合成院校列表与院校×专业叶子（用行内自建的身份归并器）。
// 详见 AggregateLeavesR。
func AggregateLeaves(rows []MajorScoreRow) ([]School, []MajorLeaf) {
	return AggregateLeavesR(rows, BuildSchoolResolver(IdentRowsFromScores(rows)))
}

// SchoolsOf 把归并器里的全部实体投影成院校列表（升序，Key 主键 + 代表代号 + 规范名）。
// 用归并器（可含计划∪分数的并集）即可让「只在计划里出现的新招生校」也进院校全集（ADR-0021）。
func SchoolsOf(r *SchoolResolver) []School {
	out := make([]School, 0, len(r.Entities()))
	for _, e := range r.Entities() {
		out = append(out, School{Key: e, Code: r.RepCode(e), Name: r.Name(e)})
	}
	return out
}

// AggregateLeavesR 用给定归并器把行表聚合成院校列表与院校×专业叶子（ADR-0021）。
// 院校列表 = 归并器全部实体（并集）；叶子键 = 院校实体键 + 渠道代表代号 + 归一化专业名，
// 故跨老/新高考换号的历史合成一条、而普通/专项两渠道分开。同一叶子同年同科类多行取最低位次
// （最难那条）。无位次省（西藏「只有分数」）退化为取最低分代表、并记分数跨度上界于 MaxScore。
func AggregateLeavesR(rows []MajorScoreRow, r *SchoolResolver) ([]School, []MajorLeaf) {
	type ypoint struct {
		ys      YearScore
		selKe   string
		selYear int
	}
	leafYears := map[string]map[string]*ypoint{} // leafID -> "year|track" -> point
	leafMeta := map[string]*MajorLeaf{}

	for _, row := range rows {
		if row.SchoolCode == "" {
			continue
		}
		ent := r.Entity(row.SchoolName)
		ch := r.Channel(row.SchoolName, row.SchoolCode)
		key := MajorKey(row.MajorName)
		leafID := ent + "/" + ch + "/" + key
		if leafMeta[leafID] == nil {
			leafMeta[leafID] = &MajorLeaf{
				SchoolKey:  ent,
				SchoolCode: ch,
				MajorKey:   key,
				MajorName:  NormalizeMajorName(row.MajorName),
			}
			leafYears[leafID] = map[string]*ypoint{}
		}
		// 最新年份的选科要求作为叶子选科
		if row.Year >= leafMeta[leafID].yearSeen {
			leafMeta[leafID].yearSeen = row.Year
			leafMeta[leafID].SelKe = row.SelKe
		}
		yk := fmt.Sprintf("%d|%s", row.Year, row.Track)
		cur := leafYears[leafID][yk]
		cand := YearScore{Year: row.Year, Track: row.Track, MinScore: row.MinScore, MinRank: row.MinRank, MaxScore: row.MaxScore}
		switch {
		case cur == nil:
			leafYears[leafID][yk] = &ypoint{ys: cand}
		case cur.ys.MinRank > 0 || row.MinRank > 0:
			// 有位次省：维持原行为——同年同科类取最低位次（最难）那条。无位次记 +∞、不会胜出。
			if rankOrInf(row.MinRank) < rankOrInf(cur.ys.MinRank) {
				leafYears[leafID][yk] = &ypoint{ys: cand}
			}
		default:
			// 都无位次（西藏「只有分数」，且 A 类/B 类两线未区分常致同键多分数）：代表取最低分（最易达
			// 的线即入），并把该 年×科类 的分数跨度上界记进 MaxScore，供叶子页展示「录取分 最低–最高」。
			lo := cur.ys.MinScore
			if row.MinScore < lo {
				lo = row.MinScore
			}
			hi := maxInt(cur.ys.MinScore, cur.ys.MaxScore, row.MinScore, row.MaxScore)
			cur.ys = YearScore{Year: row.Year, Track: row.Track, MinScore: lo, MinRank: 0, MaxScore: hi}
		}
	}

	schools := SchoolsOf(r)

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
		if leaves[i].SchoolKey != leaves[j].SchoolKey {
			return leaves[i].SchoolKey < leaves[j].SchoolKey
		}
		if leaves[i].MajorName != leaves[j].MajorName {
			return leaves[i].MajorName < leaves[j].MajorName
		}
		return leaves[i].SchoolCode < leaves[j].SchoolCode
	})
	return schools, leaves
}
