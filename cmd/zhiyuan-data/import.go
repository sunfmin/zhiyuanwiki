package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/sunfmin/zhiyuanwiki/internal/ah"
	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/hlj"
	"github.com/sunfmin/zhiyuanwiki/internal/hn"
	"github.com/sunfmin/zhiyuanwiki/internal/js"
	"github.com/sunfmin/zhiyuanwiki/internal/sc"
	"github.com/sunfmin/zhiyuanwiki/internal/store"
)

// importDefaultSrc 是新数据源（各省份/ 干净树）的默认根。见 ADR-0014。
func importDefaultSrc() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Downloads", "高考志愿", "各省份")
}

// provDirName 是某省在 各省份/ 下的子目录名（多数=中文名；个别带后缀，按需登记）。
var provDirName = map[string]string{
	"js":  "江苏",
	"hn":  "湖南",
	"sc":  "四川",
	"ah":  "安徽",
	"hlj": "黑龙江", // 各省份/黑龙江（仅一分一段从这里取；分数/计划走万师兄树，见 importHLJ）
}

// provParser 是某省入库所需的三个解析函数（签名一致，实现在 internal/<省>）。这张表是
// 「构建期 staging 管线」省份的注册处：在表中 = 走 import→DB→投影，fenduan/yuanxiao 据此分流。
// 逐行解析仍各在各省的包（ADR-0013 有意为之），这里只是选包派发，不是共用解析配置。
type provParser struct {
	Scores func(path string) ([]core.MajorScoreRow, error)
	Plan   func(path string) ([]core.PlanRow, error)
	YFD    func(path, province string, year int) ([]*core.YiFenYiDuan, error)
	// PlanMust 覆盖招生计划文件的（整路径）子串匹配；nil=默认 ["招生计划"]。个别省没有统一的
	// 「全国高校在X的招生计划」合表、且老文理副本体积更大，需精确指向该省 2025 计划文件。
	PlanMust []string
}

var provParsers = map[string]provParser{
	"js": {Scores: js.ParseScores, Plan: js.ParsePlan, YFD: js.ParseYiFenYiDuan},
	"hn": {Scores: hn.ParseScores, Plan: hn.ParsePlan, YFD: hn.ParseYiFenYiDuan},
	"sc": {Scores: sc.ParseScores, Plan: sc.ParsePlan, YFD: sc.ParseYiFenYiDuan,
		PlanMust: []string{"2025年-招生计划"}},
	"ah": {Scores: ah.ParseScores, Plan: ah.ParsePlan, YFD: ah.ParseYiFenYiDuan,
		PlanMust: []string{"安徽-2025-招生计划"}},
	// 黑龙江也是 staging 省（组模型，走 buildDBBundle），但源异构、跨两棵树，入库走 importHLJ
	// 而非通用 importProvince；这里登记是给 fenduan/yuanxiao 的「是否 staging」分流用。
	"hlj": {Scores: hlj.ParseMajorScoresXLSX, Plan: hlj.ParsePlanXLSX, YFD: hlj.ParseYiFenYiDuan},
}

// importCmd 把官方 xlsx 解析入 SQLite staging（按省幂等）。全国表（院校属性/专业门类）每次刷新。
func importCmd(args []string) {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	src := fs.String("src", importDefaultSrc(), "新数据源根目录（各省份/）")
	dbPath := fs.String("db", filepath.Join("out", "zhiyuan.db"), "SQLite staging 库路径")
	provSlug := fs.String("prov", "js", "省份 slug")
	skipNational := fs.Bool("skip-national", false, "跳过全国表刷新（批量导入多省时第二省起可加）")
	_ = fs.Parse(args)
	p := mustProv(*provSlug)

	if err := os.MkdirAll(filepath.Dir(*dbPath), 0o755); err != nil {
		fatal(err)
	}
	db, err := store.Open(*dbPath)
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	if !*skipNational {
		importNational(db, *src)
	}

	// 黑龙江/浙江源异构、布局特殊，走专用 import（见 ADR-0014 / issue #19、#20）；其余走通用 importProvince。
	switch p.slug {
	case "hlj":
		importHLJ(db, *src, p)
		return
	case "zj":
		importZJ(db, p)
		return
	}
	parser, ok := provParsers[p.slug]
	if !ok {
		fatal(fmt.Errorf("import 暂未支持省份 %q（先写 internal/%s 解析器并登记 provParsers）", p.slug, p.slug))
	}
	importProvince(db, *src, p, parser)
}

// importNational 刷新全国院校属性 + 专业门类（全量替换）。文件在任一省的 college_data 下，取首个。
func importNational(db *store.DB, src string) {
	if f := findFile(src, []string{"全国高等院校信息汇总"}, nil); f != "" {
		infos := parseSchoolInfo(f)
		if err := db.ReplaceSchools(infos); err != nil {
			fatal(err)
		}
		fmt.Printf("  全国院校属性：%d 所 → school\n", len(infos))
	} else {
		fmt.Fprintln(os.Stderr, "⚠ 未找到 全国高等院校信息汇总.xlsx，跳过院校属性")
	}
	if f := findFile(src, []string{"全国高等院校开设专业汇总"}, nil); f != "" {
		rows := parseCatalog(f)
		if err := db.ReplaceCatalog(rows); err != nil {
			fatal(err)
		}
		fmt.Printf("  全国专业门类：%d 条 → major_catalog\n", len(rows))
	} else {
		fmt.Fprintln(os.Stderr, "⚠ 未找到 全国高等院校开设专业汇总.xlsx，跳过专业门类")
	}
}

// importProvince 解析某省 专业录取分数/招生计划/一分一段 → DB（按省幂等）。文件用子树 glob
// 按名定位（路径嵌套层数因省而异，见 ADR-0014），逐行解析委派给该省 provParser。
func importProvince(db *store.DB, src string, p province, parser provParser) {
	root := filepath.Join(src, provDirName[p.slug])
	tracks := strings.Join(p.tracks, "/")

	scorePath := findFile(root, []string{"专业录取分数"}, []string{"艺术", "艺考"})
	if scorePath == "" {
		fatal(fmt.Errorf("%s：未找到 专业录取分数 xlsx（在 %s 下）", p.name, root))
	}
	scores, err := parser.Scores(scorePath)
	if err != nil {
		fatal(err)
	}
	if err := db.ReplaceScores(p.slug, scores); err != nil {
		fatal(err)
	}
	fmt.Printf("  专业录取分数：%d 行（%s·本科·含位次）→ major_score\n", len(scores), tracks)

	planMust := parser.PlanMust
	if planMust == nil {
		planMust = []string{"招生计划"}
	}
	if planPath := findFile(root, planMust, []string{"艺术", "艺考"}); planPath != "" {
		plan, err := parser.Plan(planPath)
		if err != nil {
			fatal(err)
		}
		if err := db.ReplacePlan(p.slug, plan); err != nil {
			fatal(err)
		}
		fmt.Printf("  招生计划：%d 行 → plan\n", len(plan))
	} else {
		fmt.Fprintf(os.Stderr, "⚠ 未找到%s招生计划，跳过（报考视图将为空）\n", p.name)
	}

	var allYfd []*core.YiFenYiDuan
	seenYT := map[core.YearTrack]bool{} // 防重复源文件（如四川/安徽 2025 一分一段有两份副本）
	for _, yf := range findFiles(root, []string{"一分一段表"}, []string{"艺术", "艺考"}) {
		year := yearFromName(filepath.Base(yf))
		if year == 0 {
			continue
		}
		yds, err := parser.YFD(yf, p.name, year)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ 一分一段 %s 解析失败：%v\n", filepath.Base(yf), err)
			continue
		}
		for _, y := range yds {
			yt := core.YearTrack{Year: y.Year, Track: y.Track}
			if seenYT[yt] {
				continue // 已有该 年×科类（findFiles 按体积降序，保留首个=最全副本）
			}
			seenYT[yt] = true
			allYfd = append(allYfd, y)
		}
	}
	if err := db.ReplaceYiFenYiDuan(p.slug, allYfd); err != nil {
		fatal(err)
	}
	fmt.Printf("  一分一段：%d 个(年×科类) → yifenyiduan\n", len(allYfd))
	fmt.Printf("✓ %s入库完成 → %s\n", p.name, "out/zhiyuan.db")
}

// ── 全国表解析（非省份专属，就近放 import 编排处，复用 core.OpenSheet）──

func parseSchoolInfo(path string) []store.SchoolInfo {
	s, err := core.OpenSheet(path, func(r []string) bool {
		return core.HasCell(r, "学校") && core.HasCell(r, "所在省")
	})
	if err != nil {
		fatal(err)
	}
	col := s.Col
	cName, cProv, cCity := col("学校"), col("所在省"), col("所在城市")
	cOwner, cKind := col("办学性质"), col("学校类型")
	c985, c211, cSyl := col("985"), col("211"), col("双一流")
	cRank := col("综合排名")
	var out []store.SchoolInfo
	for _, r := range s.Data {
		name := strings.TrimSpace(core.Cell(r, cName))
		if name == "" {
			continue
		}
		prov := normProvince(core.Cell(r, cProv))
		rank, _ := core.ParseLeadingInt(core.Cell(r, cRank))
		out = append(out, store.SchoolInfo{
			Name:      name,
			Province:  prov,
			City:      core.NormCity(prov, core.Cell(r, cCity)),
			Ownership: strings.TrimSpace(core.Cell(r, cOwner)),
			Kind:      strings.TrimSpace(core.Cell(r, cKind)),
			Is985:     strings.TrimSpace(core.Cell(r, c985)) == "1",
			Is211:     strings.TrimSpace(core.Cell(r, c211)) == "1",
			Syl:       strings.TrimSpace(core.Cell(r, cSyl)) == "1",
			Rank:      rank,
		})
	}
	return out
}

func parseCatalog(path string) []store.CatalogRow {
	s, err := core.OpenSheet(path, func(r []string) bool {
		return core.HasCell(r, "学校") && core.HasCell(r, "学科门类")
	})
	if err != nil {
		fatal(err)
	}
	col := s.Col
	cName, cMajor, cMenlei := col("学校"), col("专业名称", "专业"), col("学科门类", "门类")
	var out []store.CatalogRow
	for _, r := range s.Data {
		name := strings.TrimSpace(core.Cell(r, cName))
		major := strings.TrimSpace(core.Cell(r, cMajor))
		if name == "" || major == "" {
			continue
		}
		out = append(out, store.CatalogRow{
			SchoolName: name, Major: major, Menlei: strings.TrimSpace(core.Cell(r, cMenlei)),
		})
	}
	return out
}

// normProvince 把「江苏省/北京市/内蒙古自治区」归一成站点口径裸名（江苏/北京/内蒙古）。
func normProvince(s string) string {
	s = strings.TrimSpace(s)
	special := map[string]string{
		"内蒙古自治区": "内蒙古", "广西壮族自治区": "广西", "宁夏回族自治区": "宁夏",
		"新疆维吾尔自治区": "新疆", "西藏自治区": "西藏",
		"香港特别行政区": "香港", "澳门特别行政区": "澳门",
	}
	if v, ok := special[s]; ok {
		return v
	}
	s = strings.TrimSuffix(s, "省")
	s = strings.TrimSuffix(s, "市")
	return s
}

var yearRe = regexp.MustCompile(`(20\d{2})`)

func yearFromName(name string) int {
	if m := yearRe.FindString(name); m != "" {
		n, _ := core.ParseLeadingInt(m)
		return n
	}
	return 0
}

// findFiles 在 root 子树下找文件名（整路径）含全部 must、不含任一 mustNot 的 .xlsx，按体积降序。
func findFiles(root string, must, mustNot []string) []string {
	type fz struct {
		path string
		size int64
	}
	var hits []fz
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".xlsx") {
			return nil
		}
		if strings.HasPrefix(filepath.Base(path), "~$") {
			return nil // Excel 临时锁文件
		}
		for _, m := range must {
			if !strings.Contains(path, m) {
				return nil
			}
		}
		for _, m := range mustNot {
			if strings.Contains(path, m) {
				return nil
			}
		}
		info, e := d.Info()
		if e != nil {
			return nil
		}
		hits = append(hits, fz{path, info.Size()})
		return nil
	})
	sort.Slice(hits, func(i, j int) bool { return hits[i].size > hits[j].size })
	out := make([]string, len(hits))
	for i, h := range hits {
		out[i] = h.path
	}
	return out
}

// findFile 返回最佳（最大）匹配，无则空串。
func findFile(root string, must, mustNot []string) string {
	if fs := findFiles(root, must, mustNot); len(fs) > 0 {
		return fs[0]
	}
	return ""
}
