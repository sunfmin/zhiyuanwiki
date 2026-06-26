package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunfmin/zhiyuanwiki/internal/hlj"
)

// schoolDetail 是写给 Astro 的每校详情：院校 + 其全部院校×专业叶子 + 2026 报考视图。
type schoolDetail struct {
	hlj.School
	Leaves     []hlj.MajorLeaf  `json:"leaves"`
	Groups2026 []hlj.Group2026 `json:"groups2026"`
}

// trackRange 是某科类（物理/历史）在该校最新有数据年份的录取线区间。
// MaxScore↔MinRank 是最难专业（分最高、位次最靠前）；MinScore↔MaxRank 是最易专业。
// 物理 / 历史各自一套全省排名，故每科类单独成区间、不跨类混。
type trackRange struct {
	Year     int `json:"year"`
	MinScore int `json:"minScore"`
	MaxScore int `json:"maxScore"`
	MinRank  int `json:"minRank"`
	MaxRank  int `json:"maxRank"`
}

// schoolIndexEntry 是 schools.json 索引项。物理 / 历史录取线区间分开列，
// 列表可按任一科类的 MaxScore（顶尖专业录取分）排序。
type schoolIndexEntry struct {
	Code      string      `json:"code"`
	Name      string      `json:"name"`
	LeafCount int         `json:"leafCount"`
	Wuli      *trackRange `json:"wuli,omitempty"`  // 物理类
	Lishi     *trackRange `json:"lishi,omitempty"` // 历史类

	// 院校层次标签（来自万师兄旧格式表，院校级属性）。
	Is985         bool `json:"is985,omitempty"`
	Is211         bool `json:"is211,omitempty"`
	IsShuangYiLiu bool `json:"isShuangYiLiu,omitempty"`
}

// schoolMetaOut 是 public/data/school-meta.json 的一条（按院校代码建键），承载位次定位结果过滤
// 用的院校级属性。紧凑键、空值省略；客户端一次性 fetch、按 sc 挂接。见 ADR-0008。
type schoolMetaOut struct {
	Province string   `json:"p,omitempty"`  // 省份
	City     string   `json:"c,omitempty"`  // 城市
	CityTier string   `json:"ct,omitempty"` // 城市层级
	Owner    string   `json:"o,omitempty"`  // 办学性质
	Kind     string   `json:"k,omitempty"`  // 学校类别
	Levels   []string `json:"lv,omitempty"` // 层次：["985","211","双一流"] 中为真者
}

// tagSourceFiles 列出含「_985/_211/双一流」列的旧格式文件（相对数据根目录）。
// 跨年合并以最大化覆盖（标签是院校级属性，与年份无关；缺失项静默跳过）。
func tagSourceFiles(src string) []string {
	rel := []string{
		"24-万师兄-黑龙江2026年高考志愿填报大数据/01-万师兄-黑龙江高考-招生计划-2020-2026/黑龙江_招生计划_2020.xlsx",
		"24-万师兄-黑龙江2026年高考志愿填报大数据/01-万师兄-黑龙江高考-招生计划-2020-2026/黑龙江_招生计划_2021.xlsx",
		"24-万师兄-黑龙江2026年高考志愿填报大数据/01-万师兄-黑龙江高考-招生计划-2020-2026/黑龙江_招生计划_2022.xlsx",
		"24-万师兄-黑龙江2026年高考志愿填报大数据/01-万师兄-黑龙江高考-招生计划-2020-2026/黑龙江_招生计划_2023.xlsx",
		"24-万师兄-黑龙江2026年高考志愿填报大数据/04-万师兄-黑龙江高考-专业录取分数线-2020-2025/黑龙江_专业分数线_2020.xlsx",
		"24-万师兄-黑龙江2026年高考志愿填报大数据/04-万师兄-黑龙江高考-专业录取分数线-2020-2025/黑龙江_专业分数线_2021.xlsx",
		"24-万师兄-黑龙江2026年高考志愿填报大数据/03-万师兄-黑龙江高考-投档线-2020-2025/黑龙江_投档线_2020.xlsx",
		"24-万师兄-黑龙江2026年高考志愿填报大数据/03-万师兄-黑龙江高考-投档线-2020-2025/黑龙江_投档线_2021.xlsx",
		"24-万师兄-黑龙江2026年高考志愿填报大数据/03-万师兄-黑龙江高考-投档线-2020-2025/黑龙江_投档线_2022.xlsx",
	}
	out := make([]string, len(rel))
	for i, r := range rel {
		out[i] = filepath.Join(src, filepath.FromSlash(r))
	}
	return out
}

// rangeForTrack 汇总某科类在该校最新有数据年份的录取线区间；该科类无数据返回 nil。
func rangeForTrack(leaves []hlj.MajorLeaf, track string) *trackRange {
	year := 0
	for _, lf := range leaves {
		for _, ys := range lf.Years {
			if ys.Track == track && ys.MinScore > 0 && ys.Year > year {
				year = ys.Year
			}
		}
	}
	if year == 0 {
		return nil
	}
	r := &trackRange{Year: year}
	for _, lf := range leaves {
		for _, ys := range lf.Years {
			if ys.Track != track || ys.Year != year || ys.MinScore <= 0 {
				continue
			}
			if r.MinScore == 0 || ys.MinScore < r.MinScore {
				r.MinScore = ys.MinScore
			}
			if ys.MinScore > r.MaxScore {
				r.MaxScore = ys.MinScore
			}
			if ys.MinRank > 0 {
				if r.MinRank == 0 || ys.MinRank < r.MinRank {
					r.MinRank = ys.MinRank
				}
				if ys.MinRank > r.MaxRank {
					r.MaxRank = ys.MinRank
				}
			}
		}
	}
	return r
}

// yuanxiaoCmd 解析专业录取分数线（新科类年份）→ 院校 / 院校×专业叶子 JSON。
func yuanxiaoCmd(args []string) {
	fs := flag.NewFlagSet("yuanxiao", flag.ExitOnError)
	src := fs.String("src", defaultSrc(), "官方数据根目录")
	out := fs.String("out", filepath.Join("src", "data"), "JSON 输出目录")
	pub := fs.String("pub", filepath.Join("public", "data"), "客户端公开数据目录（school-meta.json）")
	_ = fs.Parse(args)

	scoreDir := filepath.Join(*src,
		"24-万师兄-黑龙江2026年高考志愿填报大数据",
		"04-万师兄-黑龙江高考-专业录取分数线-2020-2025")

	years := []int{2023, 2024, 2025} // 新科类（物理/历史 + 位次），见 ADR-0007
	var all []hlj.MajorScoreRow
	for _, y := range years {
		path := findScoreFile(scoreDir, y)
		if path == "" {
			fmt.Fprintf(os.Stderr, "跳过 %d：未找到文件\n", y)
			continue
		}
		rows, err := hlj.ParseMajorScoresXLSX(path)
		if err != nil {
			fatal(err)
		}
		fmt.Printf("  %d：%d 行（物理/历史·本科批·含位次）\n", y, len(rows))
		all = append(all, rows...)
	}
	if len(all) == 0 {
		fatal(fmt.Errorf("未解析到任何专业分数线行"))
	}

	schools, leaves := hlj.AggregateLeaves(all)

	// 院校属性（省份/城市/办学性质/学校类别 + 985/211/双一流）与 专业名→学科门类 映射，
	// 均取自万师兄旧格式表（按校名挂接）。见 ADR-0008。
	tagFiles := tagSourceFiles(*src)
	meta := hlj.LoadSchoolMeta(tagFiles)
	menlei := hlj.LoadMenlei(tagFiles)
	fmt.Printf("  院校属性库 %d 所 · 专业→门类精确映射 %d 条\n", meta.Len(), menlei.Len())

	// 一分一段总人数（用于等效位次缩放）。目前仅 2026 物理为 .xlsx 可读；
	// 历史年份为 .xls，待接入后等效位次方能跨年缩放，否则回退原位次。
	totals := map[hlj.YearTrack]int{}
	if y, err := hlj.ParseYiFenYiDuanXLSX(
		filepath.Join(*src, "黑龙江2026物理类一分一段表.xlsx"), "黑龙江", "物理", 2026); err == nil {
		totals[hlj.YearTrack{Year: 2026, Track: "物理"}] = y.Total()
	}

	// 2026 招生计划 → 院校专业组单年视图，按院校+专业名挂接往年位次（ADR-0003）。
	planPath := filepath.Join(*src,
		"24-万师兄-黑龙江2026年高考志愿填报大数据",
		"01-万师兄-黑龙江高考-招生计划-2020-2026", "黑龙江_招生计划_2026.xlsx")
	var groupsByCode map[string][]hlj.Group2026
	if planRows, err := hlj.ParsePlanXLSX(planPath); err != nil {
		fmt.Fprintf(os.Stderr, "警告：未读到 2026 招生计划（%v），跳过组视图\n", err)
		groupsByCode = map[string][]hlj.Group2026{}
	} else {
		fmt.Printf("  2026 招生计划：%d 行（物理/历史·本科批）\n", len(planRows))
		groupsByCode = hlj.BuildGroups2026(planRows, leaves, totals, menlei.Code)
	}

	// 按院校分组叶子
	byCode := map[string][]hlj.MajorLeaf{}
	for _, lf := range leaves {
		byCode[lf.SchoolCode] = append(byCode[lf.SchoolCode], lf)
	}

	// schools.json 索引 + school-meta.json 过滤属性表（均按院校名挂接 meta）。
	index := make([]schoolIndexEntry, 0, len(schools))
	metaOut := make(map[string]schoolMetaOut, len(schools))
	tagged, n985, n211, nSyl := 0, 0, 0, 0
	for _, s := range schools {
		lvs := byCode[s.Code]
		e := schoolIndexEntry{
			Code: s.Code, Name: s.Name, LeafCount: len(lvs),
			Wuli:  rangeForTrack(lvs, "物理"),
			Lishi: rangeForTrack(lvs, "历史"),
		}
		if m, ok := meta.Lookup(s.Name); ok {
			e.Is985, e.Is211, e.IsShuangYiLiu = m.Is985, m.Is211, m.IsShuangYiLiu
			tagged++
			if m.Is985 {
				n985++
			}
			if m.Is211 {
				n211++
			}
			if m.IsShuangYiLiu {
				nSyl++
			}
			metaOut[s.Code] = schoolMetaOut{
				Province: m.Province, City: m.City, CityTier: hlj.CityTier(m.City),
				Owner: m.Ownership, Kind: m.Kind, Levels: m.Levels(),
			}
		}
		index = append(index, e)
	}
	fmt.Printf("  院校属性挂接 %d/%d 所（985=%d 211=%d 双一流=%d）\n",
		tagged, len(schools), n985, n211, nSyl)
	writeJSON(filepath.Join(*out, "schools.json"), index)

	if err := os.MkdirAll(*pub, 0o755); err != nil {
		fatal(err)
	}
	writeJSON(filepath.Join(*pub, "school-meta.json"), metaOut)
	fmt.Printf("  院校过滤属性：%d 所 → %s\n", len(metaOut), filepath.Join(*pub, "school-meta.json"))

	// 每校详情
	detailDir := filepath.Join(*out, "schools")
	if err := os.MkdirAll(detailDir, 0o755); err != nil {
		fatal(err)
	}
	groupCount := 0
	for _, s := range schools {
		d := schoolDetail{School: s, Leaves: byCode[s.Code], Groups2026: groupsByCode[s.Code]}
		groupCount += len(d.Groups2026)
		writeJSON(filepath.Join(detailDir, s.Code+".json"), d)
	}

	fmt.Printf("✓ 院校 %d 所 · 院校×专业叶子 %d 个 · 2026 院校专业组 %d 个 → %s\n",
		len(schools), len(leaves), groupCount, *out)
}

// findScoreFile 在目录里找某年的专业分数线文件（连字符或下划线命名）。
func findScoreFile(dir string, year int) string {
	for _, name := range []string{
		fmt.Sprintf("黑龙江-专业分数线-%d.xlsx", year),
		fmt.Sprintf("黑龙江_专业分数线_%d.xlsx", year),
	} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
