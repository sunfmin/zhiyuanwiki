package core

import (
	"fmt"
	"sort"
)

// ── 院校身份归并（ADR-0021）─────────────────────────────────────────────
//
// 院校代号逐年逐源不稳（老/新高考换本、相邻年插校位移、同校多代号、跨年复用），不能当主键。
// SchoolResolver 把 (代号,校名,年份) 观测归并成两把稳定的键：
//
//   - 实体键 entityKey = 归一化校名（NormName）——本站院校主键（报考实体粒度，保留校区后缀）。
//   - 渠道键 channelKey = 同一实体内的招生渠道（普通/专项/中外…）的代表代号。判据：同实体内
//     「年份集合不相交」的代号视作同一渠道（老/新高考换本，如深圳 2046@2023 与 2044@2024-25），
//     「同年并存」的代号视作不同渠道（如黑龙江大学普通 1408 与专项 6001）。这样叶子既不会把改本
//     换号的历史劈成两段，也不会把普通/专项两条录取线塌成一条。
//
// 改名（同代号、相邻年、校名共享前缀）并入最新名的实体（湖州师范学院→大学）；跨年复用同一代号
// 但校名无关且不相邻的（深圳↔广州 皆用过 2046）不并、按各自校名分成两个实体。

// IdentRow 是身份归并所需的最小观测：一条 (代号,校名,年份)，来自招生计划或录取分数任一。
type IdentRow struct {
	Code string
	Name string
	Year int
}

// SchoolResolver 见文件头注释。用 BuildSchoolResolver 构造，只读。
type SchoolResolver struct {
	entityOf  map[string]string // NormName(原始校名) -> 归并后实体键（改名并入最新名）
	channelOf map[string]string // entityKey|code -> 渠道代表代号
	nameOf    map[string]string // entityKey -> 规范展示名（最新年）
	repOf     map[string]string // entityKey -> 代表代号（最新年，同年取最小）
	entities  []string          // 升序实体键
	renames   []string          // 归并日志（人工复核用）：改名/转设
}

// Entity 返回某原始校名对应的稳定实体键（已应用改名归并）。未知名回退到其 NormName。
func (r *SchoolResolver) Entity(name string) string {
	if e, ok := r.entityOf[NormName(name)]; ok {
		return e
	}
	return NormName(name)
}

// Channel 返回 (校名,代号) 所属渠道的代表代号；用于把同校多渠道的叶子分开。
func (r *SchoolResolver) Channel(name, code string) string {
	if c, ok := r.channelOf[r.Entity(name)+"|"+code]; ok {
		return c
	}
	return code
}

// Name 返回实体的规范展示名（最新年）。
func (r *SchoolResolver) Name(entity string) string { return r.nameOf[entity] }

// RepCode 返回实体的代表代号（最新年、同年最小）——供展示「院校代码」，非主键。
func (r *SchoolResolver) RepCode(entity string) string { return r.repOf[entity] }

// Entities 返回全部实体键（升序）。
func (r *SchoolResolver) Entities() []string { return r.entities }

// Renames 返回改名/转设归并日志（人工复核用）。
func (r *SchoolResolver) Renames() []string { return r.renames }

// sharePrefix 判断两校名是否共享 >=2 前导字（改名/升格/转设的稳定信号：湖州师范/安徽科技/大连…
// 都保留前导地名或专名；深圳↔广州 前导不同，不会误并）。按 rune 计。
func sharePrefix(a, b string) bool {
	ra, rb := []rune(a), []rune(b)
	n := 0
	for n < len(ra) && n < len(rb) && ra[n] == rb[n] {
		n++
	}
	return n >= 2
}

// BuildSchoolResolver 从计划+分数的全部 (代号,校名,年份) 观测构造归并器。
func BuildSchoolResolver(rows []IdentRow) *SchoolResolver {
	// 1) 每代号：各年份→校名（NormName），及年份集合。
	type codeInfo struct {
		yearName map[int]string // year -> NormName(name)
		years    map[int]bool
	}
	codes := map[string]*codeInfo{}
	for _, r := range rows {
		if r.Code == "" || r.Name == "" {
			continue
		}
		ci := codes[r.Code]
		if ci == nil {
			ci = &codeInfo{yearName: map[int]string{}, years: map[int]bool{}}
			codes[r.Code] = ci
		}
		// 同代号同年多行取任一名（同年同代号名一致）
		ci.yearName[r.Year] = NormName(r.Name)
		ci.years[r.Year] = true
	}

	// 2) 改名归并：同代号、相邻年、校名变化且共享前缀 → 归并到最新名。用并查集按 NormName 名归并。
	uf := newUF()
	canonYear := map[string]int{} // NormName名 -> 其最新出现年（决定并查集里谁当规范名）
	rawName := map[string]string{}
	for _, r := range rows {
		if r.Name == "" {
			continue
		}
		nn := NormName(r.Name)
		rawName[nn] = r.Name
		if r.Year > canonYear[nn] {
			canonYear[nn] = r.Year
		}
	}
	// 换本边界识别：老→新高考改革那年，整本院校代号重排，同一代号在相邻两年指向无关的两所学校
	//（内蒙古农业大学↔内蒙古科技大学包头医学院…）。这类「同代号相邻年异名」是代号复用、不是改名，
	// 但会共享地名前导（北京/南京/内蒙古…）被误并。特征：该年对里「共用代号且校名变了」的比例极高
	//（改革年 ~25%，常态 <3%）。故先把高换手率的相邻年对标为换本边界，改名识别一律跳过这些边界。
	pairShared := map[int]int{} // y0 -> 该年对(y0,y0+1)共用代号数
	pairChanged := map[int]int{}
	for _, ci := range codes {
		for y0 := range ci.years {
			if ci.years[y0+1] {
				pairShared[y0]++
				if ci.yearName[y0] != ci.yearName[y0+1] {
					pairChanged[y0]++
				}
			}
		}
	}
	renumberBoundary := map[int]bool{} // y0 -> (y0,y0+1) 是换本边界
	for y0, shared := range pairShared {
		if shared >= 20 && float64(pairChanged[y0])/float64(shared) > 0.2 {
			renumberBoundary[y0] = true
		}
	}

	var renameLog []string
	for code, ci := range codes {
		yrs := sortedYears(ci.years)
		for i := 1; i < len(yrs); i++ {
			y0, y1 := yrs[i-1], yrs[i]
			n0, n1 := ci.yearName[y0], ci.yearName[y1]
			if n0 == n1 {
				continue
			}
			// 相邻年 + 共享前导 + 非换本边界 = 改名/升格/转设
			if y1-y0 == 1 && !renumberBoundary[y0] && sharePrefix(n0, n1) {
				uf.union(n0, n1)
				renameLog = append(renameLog,
					fmt.Sprintf("改名归并 代号%s：%s(%d) → %s(%d)", code, n0, y0, n1, y1))
			}
		}
	}

	// 3) 每个 NormName 名 → 实体键（并查集根里最新年的名）。
	entityOf := map[string]string{}
	for nn := range rawName {
		root := uf.find(nn)
		// 取该并查集集合里 canonYear 最大的名当规范实体键
		entityOf[nn] = nn // 先自指
		_ = root
	}
	// 计算每个集合的规范名（最新年）
	setCanon := map[string]string{} // root -> canonical NormName
	for nn := range rawName {
		root := uf.find(nn)
		if c, ok := setCanon[root]; !ok || canonYear[nn] > canonYear[c] {
			setCanon[root] = nn
		}
	}
	for nn := range rawName {
		entityOf[nn] = setCanon[uf.find(nn)]
	}

	// 4) 实体的规范展示名与代表代号；并把每个代号的行按实体归属（同代号可能跨实体，如复用 2046）。
	//    收集 entity -> (code -> years) 供渠道划分。
	nameOf := map[string]string{}
	entYear := map[string]int{} // entity -> 该实体最新年
	entCodeYears := map[string]map[string]map[int]bool{}
	for code, ci := range codes {
		for y, nn := range ci.yearName {
			ent := entityOf[nn]
			if _, ok := entCodeYears[ent]; !ok {
				entCodeYears[ent] = map[string]map[int]bool{}
			}
			if _, ok := entCodeYears[ent][code]; !ok {
				entCodeYears[ent][code] = map[int]bool{}
			}
			entCodeYears[ent][code][y] = true
			if y >= entYear[ent] {
				entYear[ent] = y
				nameOf[ent] = rawName[nn]
			}
		}
	}

	// 5) 渠道划分（每实体）：把代号按「年份不相交=同渠道」贪心归并。种子顺序：最新年降序、
	//    再按覆盖年份数降序（让主渠道普通先成种子，老高考码并入普通而非专项）。
	channelOf := map[string]string{}
	repOf := map[string]string{}
	for ent, cy := range entCodeYears {
		type cinfo struct {
			code   string
			years  map[int]bool
			latest int
			span   int
		}
		var cs []cinfo
		for code, ys := range cy {
			latest, span := 0, len(ys)
			for y := range ys {
				if y > latest {
					latest = y
				}
			}
			cs = append(cs, cinfo{code, ys, latest, span})
		}
		sort.Slice(cs, func(i, j int) bool {
			if cs[i].latest != cs[j].latest {
				return cs[i].latest > cs[j].latest
			}
			if cs[i].span != cs[j].span {
				return cs[i].span > cs[j].span
			}
			return cs[i].code < cs[j].code
		})
		type channel struct {
			rep   string
			years map[int]bool
		}
		var chans []*channel
		for _, c := range cs {
			var joined *channel
			for _, ch := range chans {
				if disjoint(ch.years, c.years) {
					joined = ch
					break
				}
			}
			if joined == nil {
				joined = &channel{rep: c.code, years: map[int]bool{}}
				chans = append(chans, joined)
			}
			for y := range c.years {
				joined.years[y] = true
			}
			channelOf[ent+"|"+c.code] = joined.rep
		}
		// 代表代号 = 覆盖最新年的渠道的 rep（种子即最新年最大者，取 chans[0].rep）
		if len(chans) > 0 {
			repOf[ent] = chans[0].rep
		}
	}

	entities := make([]string, 0, len(nameOf))
	for e := range nameOf {
		entities = append(entities, e)
	}
	sort.Strings(entities)

	return &SchoolResolver{
		entityOf:  entityOf,
		channelOf: channelOf,
		nameOf:    nameOf,
		repOf:     repOf,
		entities:  entities,
		renames:   renameLog,
	}
}

func disjoint(a, b map[int]bool) bool {
	for y := range a {
		if b[y] {
			return false
		}
	}
	return true
}

func sortedYears(ys map[int]bool) []int {
	out := make([]int, 0, len(ys))
	for y := range ys {
		out = append(out, y)
	}
	sort.Ints(out)
	return out
}

// 极简并查集（按名字符串）。
type unionFind struct{ parent map[string]string }

func newUF() *unionFind { return &unionFind{parent: map[string]string{}} }
func (u *unionFind) find(x string) string {
	if _, ok := u.parent[x]; !ok {
		u.parent[x] = x
	}
	for u.parent[x] != x {
		u.parent[x] = u.parent[u.parent[x]]
		x = u.parent[x]
	}
	return x
}
func (u *unionFind) union(a, b string) {
	ra, rb := u.find(a), u.find(b)
	if ra != rb {
		u.parent[ra] = rb
	}
}

// IdentRowsFromScores / IdentRowsFromPlan 便捷构造身份观测；投影层把两者 append 起来喂给
// BuildSchoolResolver，即得「计划∪分数」的院校全集（含只在计划里出现的新招生校）。
func IdentRowsFromScores(rows []MajorScoreRow) []IdentRow {
	out := make([]IdentRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, IdentRow{Code: r.SchoolCode, Name: r.SchoolName, Year: r.Year})
	}
	return out
}

func IdentRowsFromPlan(rows []PlanRow) []IdentRow {
	out := make([]IdentRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, IdentRow{Code: r.SchoolCode, Name: r.SchoolName, Year: r.Year})
	}
	return out
}
