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

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/group3p12"
	"github.com/sunfmin/zhiyuanwiki/internal/hlj"
	"github.com/sunfmin/zhiyuanwiki/internal/store"
	"github.com/sunfmin/zhiyuanwiki/internal/tj"
	"github.com/sunfmin/zhiyuanwiki/internal/xj"
	"github.com/sunfmin/zhiyuanwiki/internal/xz"
	"github.com/sunfmin/zhiyuanwiki/internal/zj"
)

// importDefaultSrc 是通用 importProvince 的默认源根：各省份/ 干净树。见 ADR-0014。
// 与 zj/hlj 的 defaultSrc() 同源（都在 ~/Downloads/高考志愿 下），故由后者派生，避免根路径两处写。
func importDefaultSrc() string {
	return filepath.Join(defaultSrc(), "各省份")
}

// provDirName 是某省在 各省份/ 下的子目录名（多数=中文名；个别带后缀，按需登记）。
var provDirName = map[string]string{
	"js":    "江苏",
	"hn":    "湖南",
	"sc":    "四川",
	"ah":    "安徽",
	"gx":    "广西",
	"sx":    "陕西",
	"bj":    "北京",
	"sh":    "上海",
	"hain":  "海南",
	"nm":    "内蒙", // 各省份/内蒙（省名内蒙古）
	"gd":    "广东",
	"fj":    "福建",
	"nx":    "宁夏",
	"jx":    "江西",
	"jl":    "吉林",
	"gs":    "甘肃",
	"cq":    "重庆",
	"gz":    "贵州",
	"ln":    "辽宁",
	"hebei": "河北",
	"sd":    "山东",
	"zj":    "浙江", // 改用一致源 各省份/浙江（源在 …/浙江/浙江/ 下，子树 glob 命中；退役万师兄 09、浙江，见 ADR-0022）
	"tj":    "天津",
	"qh":    "青海", // 各省份/青海 下并存旧万师兄树(青海/29.青海，2017-2022)与 2、…22-25 现行数据，靠 *Must 精确指向后者
	"xj":    "新疆", // 源在 各省份/新疆/新疆/新疆/ 下（嵌套多层，靠子树 glob）
	// 西藏数据是独立交付包（不在 各省份/ 树内），用 -src ~/Downloads 指向其上层、provDirName 给包名。
	"xz": "31、西藏-2026志愿填报资料",

	"hb":     "湖北高考数据", // 各省份/ 下子目录带后缀
	"yn":     "云南",
	"henan":  "河南",  // slug 用 henan 避免与湖南 hn 冲突
	"hlj":    "黑龙江", // 各省份/黑龙江（仅一分一段从这里取；分数/计划走万师兄树，见 importHLJ）
	"shanxi": "山西",  // slug 用 shanxi 避免与陕西 sx 冲突；源异构走 importShanxi
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
	// ScoreMust 覆盖录取分数文件的子串匹配；nil=默认 ["专业录取分数"]。个别省分数文件名是「专业
	// 分数线」（内蒙/宁夏），或同名 2024 副本体积更大会被按体积错选（广东/福建），需精确指向 2025 文件。
	ScoreMust []string
}

var provParsers = map[string]provParser{
	// 统一格式 3+1+2 group 省（物理类/历史类 · 含最低位次 · 计划带院校专业组代码），逐行解析同形，
	// 全部共用 internal/group3p12（ADR-0013/0014：同构省共享一份）。江苏/湖南无需 PlanMust；四川/安徽
	// 的 25 年计划目录名含「招生计划」子串、且有更大的同名副本，需 PlanMust 精确指向。
	"js": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan},
	"hn": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan},
	"sc": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		PlanMust: []string{"2025年-招生计划"}},
	"ah": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		PlanMust: []string{"安徽-2025-招生计划"}},
	// 其余同构 group 省（广西/陕西/…）：PlanMust 必填且要精确：这些省的 25 年数据目录名常含「招生计划」
	// 子串（如「25招生计划和投档线」/「25分数线和招生计划」），默认 ["招生计划"] 会按体积错配到更大的
	// 「专业录取分数」文件。
	"gx": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		PlanMust: []string{"广西-2025-招生计划"}},
	"sx": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"25年全国高校在陕西的专业录取分数"}, PlanMust: []string{"25年全国高校在陕西的招生计划"}},
	// 综合(3+3)+组：科类「综合」已被 group3p12.keep 放行；上海组列名「专业组代码」已加别名。
	"bj": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"25年全国高校在北京的专业录取分数"}, PlanMust: []string{"2025年全国高校在北京市的招生计划"}},
	// 上海：最低分被官方封顶在 580（高分段不披露），改用未封顶的「平均分/平均位次」作录取参考分。见 group3p12.ParseScoresAvg。
	"sh": {Scores: group3p12.ParseScoresAvg, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"上海_专业分数线_2025"}, PlanMust: []string{"上海2025招生计划"}},
	"hain": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"25年全国高校在海南的专业录取分数"}, PlanMust: []string{"海南25招生计划含院校代码"}},
	// 分数文件名是「专业分数线」（内蒙/宁夏）或 2024 同名副本更大（广东/福建），用 ScoreMust 精确指向 2025。
	"nm": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"25年全国高校在内蒙古的专业分数线"}, PlanMust: []string{"20250617-内蒙古2025招生计划"}},
	"gd": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"25年全国高校在广东的专业录取分数"}, PlanMust: []string{"广东-2025年-招生计划"}},
	"fj": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"25年全国高校在福建的专业录取分数"}, PlanMust: []string{"福建2025招生计划"}},
	"nx": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"22-25年全国高校在宁夏的专业分数线"}, PlanMust: []string{"在宁夏的招生计划"}},
	// 计划表列名异形，靠 group3p12 列别名覆盖；分数文件名「专业分数线」（吉林）或精确指向 2025。
	"jx": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"25年全国高校在江西的专业录取分数"}, PlanMust: []string{"江西省2025招生计划"}},
	"jl": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"25年全国高校在吉林省的专业分数线"}, PlanMust: []string{"吉林省-2025年-招生计划"}},
	"gs": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"25年全国高校在甘肃的专业录取分数"}, PlanMust: []string{"甘肃2025年招生计划"}},
	// major 模型省（无组）——解析仍用 group3p12，build 经 model="major" 走 buildMajorBundle。
	// 分数源多为 22-25 合表（老文理年份的科类被 keep 过滤，仅 2025 物理/历史/综合 入库）。
	"cq": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"22-25年全国高校在重庆的专业录取分数"}, PlanMust: []string{"22-25年全国高校在重庆的招生计划"}},
	"gz": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"22-25年全国高校在贵州的专业录取分数"}, PlanMust: []string{"22-25年全国高校在贵州的招生计划"}},
	"ln": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"25年全国高校在辽宁的专业录取分数"}, PlanMust: []string{"在辽宁省的招生计划"}},
	"hebei": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"25年全国高校在河北省的专业分数线"}, PlanMust: []string{"2025年河北招生计划"}},
	"sd": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"25年全国高校在山东省的专业分数线"}, PlanMust: []string{"全国高校在山东省的招生计划"}},
	// 浙江：综合(3+3)·专业平行志愿(无组)·major 模型，与山东同型。改用一致源 各省份/浙江（退役万师兄
	// 09、浙江 专属栈，见 ADR-0022）。ScoreMust 精确指向「专业录取分数」以避开同目录「院校录取分数」合表；
	// 一分一段沿用 zj.ParseYiFenYiDuan（浙江综合单列格式，与通用 group3p12 表头不同）。
	"zj": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: zj.ParseYiFenYiDuan,
		ScoreMust: []string{"22-25年全国高校在浙江的专业录取分数"}, PlanMust: []string{"22-25年全国高校在浙江的招生计划"}},
	// 青海：同 cq/gz 的 22-25 合表（解析复用 group3p12，build 走 major）。各省份/青海 下另有旧万师兄树
	// （青海/29.青海/2025年…招生计划.xlsx 体积更大），用 *Must 精确指向「22-25年全国高校在青海的…」现行文件。
	"qh": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"22-25年全国高校在青海的专业录取分数"}, PlanMust: []string{"22-25年全国高校在青海的招生计划"}},
	// 天津：综合+院校专业组（group），但计划表无院校代码（由专业组代码剥后缀得）+ 无科类列，自定义 tj.ParsePlan。
	"tj": {Scores: group3p12.ParseScores, Plan: tj.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		ScoreMust: []string{"25年全国高校在天津的专业录取分数"}, PlanMust: []string{"天津-2025年-招生计划"}},
	// 新疆：老文理（理科/文科）专属 xj 解析（keep={理科,文科}，yfd 无批次列）。ScoreMust 精确指向
	// 「专业录取分数」以避开同目录的「院校录取分数」合表。计划/一分一段走默认子串。
	"xj": {Scores: xj.ParseScores, Plan: xj.ParsePlan, YFD: xj.ParseYiFenYiDuan,
		ScoreMust: []string{"22-25年全国高校在新疆的专业录取分数"}},
	// 西藏：老文理「只有分数」省（无位次/无一分一段，专属 xz 解析）。YFD 留空——源无一分一段，
	// importProvince 的「一分一段表」glob 命中不到任何文件，循环不执行、不会调用 nil。Score/Plan
	// 精确指向 22-25 合表，避开历史数据目录下的「西藏_专业分数线/西藏_招生计划_YYYY」分年小文件。
	"xz": {Scores: xz.ParseScores, Plan: xz.ParsePlan,
		ScoreMust: []string{"22-25年全国高校在西藏的专业录取分数"}, PlanMust: []string{"22-25年全国高校在西藏的招生计划"}},
	"hb": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		PlanMust: []string{"25年全国高校在湖北省的招生计划"}},
	"yn": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		// 2025 单年计划在「云南25年高考数据」目录；用目录名 + 招生计划 双串排除「22~25年…」多年合表。
		PlanMust: []string{"云南25年高考数据", "招生计划"}},
	"henan": {Scores: group3p12.ParseScores, Plan: group3p12.ParsePlan, YFD: group3p12.ParseYiFenYiDuan,
		PlanMust: []string{"河南-2025-招生计划"}},
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

	// 黑龙江/山西源异构、布局特殊，走专用 import（见 ADR-0014 / issue #19）；其余走通用 importProvince
	// （浙江自 ADR-0022 起改用一致源 各省份/浙江，回归通用路径）。
	switch p.slug {
	case "hlj":
		importHLJ(db, *src, p)
		return
	case "shanxi":
		importShanxi(db, *src, p)
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
		logSrc("全国院校属性", f)
		infos := parseSchoolInfo(f)
		if err := db.ReplaceSchools(infos); err != nil {
			fatal(err)
		}
		fmt.Printf("  全国院校属性：%d 所 → school\n", len(infos))
	} else {
		fmt.Fprintln(os.Stderr, "⚠ 未找到 全国高等院校信息汇总.xlsx，跳过院校属性")
	}
	if f := findFile(src, []string{"全国高等院校开设专业汇总"}, nil); f != "" {
		logSrc("全国专业门类", f)
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

	scoreMust := parser.ScoreMust
	if scoreMust == nil {
		scoreMust = []string{"专业录取分数"}
	}
	scorePath := findFile(root, scoreMust, []string{"艺术", "艺考"})
	if scorePath == "" {
		fatal(fmt.Errorf("%s：未找到录取分数 xlsx（%v，在 %s 下）", p.name, scoreMust, root))
	}
	logSrc("录取分数", scorePath)
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
		logSrc("招生计划", planPath)
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
		used := false
		for _, y := range yds {
			yt := core.YearTrack{Year: y.Year, Track: y.Track}
			if seenYT[yt] {
				continue // 已有该 年×科类（findFiles 按体积降序，保留首个=最全副本）
			}
			seenYT[yt] = true
			used = true
			allYfd = append(allYfd, y)
		}
		if used {
			logSrc(fmt.Sprintf("一分一段·%d", year), yf)
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
