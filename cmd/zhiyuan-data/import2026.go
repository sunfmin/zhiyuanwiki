package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/store"
)

// yfd2026File 是 workflow 抓取产物 out/2026yfd/<slug>.json 的结构：某省 2026 一分一段，
// 逐科类一张全表（entries 每分一行）。控制线 = 本科线（特控线可在 notes 另记）。
type yfd2026File struct {
	Slug     string `json:"slug"`
	Province string `json:"province"`
	Status   string `json:"status"`
	Tracks   []struct {
		Track       string              `json:"track"`
		ControlLine int                 `json:"controlLine"`
		Entries     []core.FenduanEntry `json:"entries"`
	} `json:"tracks"`
}

// import2026Cmd 把 workflow 抓到的 2026 一分一段 JSON（每省 <slug>.json）按 省×年 additive
// 写入 staging（只删/插 year=2026，不动 24/25）。入库后各省再跑 `fenduan -prov <slug>` 投影。
// 是「各省 2026 陆续发布、单独补一个新年份」的入口，区别于跑全部年份的通用 import（见 ADR-0014）。
func import2026Cmd(args []string) {
	fs := flag.NewFlagSet("import2026", flag.ExitOnError)
	src := fs.String("src", filepath.Join("out", "2026yfd"), "2026 一分一段 JSON 目录（每省 <slug>.json）")
	dbPath := fs.String("db", filepath.Join("out", "zhiyuan.db"), "SQLite staging 库")
	year := fs.Int("year", 2026, "年份")
	only := fs.String("prov", "", "只导入这个 slug（空=src 下全部 <slug>.json）")
	_ = fs.Parse(args)

	files, err := collect2026Files(*src, *only)
	if err != nil {
		fatal(err)
	}
	if len(files) == 0 {
		fatal(fmt.Errorf("%s 下没有可导入的 <slug>.json（-prov=%q）", *src, *only))
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	okN, skipN := 0, 0
	for _, path := range files {
		slug, yfds, err := parse2026File(path, *year)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ %s：%v（跳过）\n", filepath.Base(path), err)
			skipN++
			continue
		}
		if len(yfds) == 0 {
			fmt.Fprintf(os.Stderr, "⚠ %s：无可入库科类（跳过）\n", filepath.Base(path))
			skipN++
			continue
		}
		if err := db.ReplaceYiFenYiDuanYear(slug, *year, yfds); err != nil {
			fatal(err)
		}
		parts := make([]string, len(yfds))
		for i, y := range yfds {
			parts[i] = fmt.Sprintf("%s %d段(顶%d 底%d 总%d)", y.Track, len(y.Entries),
				y.Entries[len(y.Entries)-1].Score, y.Entries[0].Score, y.Total())
		}
		fmt.Printf("✓ %s %d 一分一段 → yifenyiduan：%s\n", slug, *year, strings.Join(parts, " · "))
		okN++
	}
	fmt.Printf("\n完成：入库 %d 省，跳过 %d。下一步对入库省份跑 `zhiyuan-data fenduan -prov <slug>` 投影。\n", okN, skipN)
}

// collect2026Files 列出 src 下要处理的 <slug>.json（only 非空则只那一个）。
func collect2026Files(src, only string) ([]string, error) {
	if only != "" {
		p := filepath.Join(src, only+".json")
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("找不到 %s", p)
		}
		return []string{p}, nil
	}
	ents, err := os.ReadDir(src)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		out = append(out, filepath.Join(src, e.Name()))
	}
	sort.Strings(out)
	return out, nil
}

// parse2026File 读一个 <slug>.json，校验+归一成 []*core.YiFenYiDuan（Year=year，Province=站点口径名）。
// 校验不过的科类被丢弃并告警（绝不把坏表塞进 DB）：entries 非空、累计正且随分数升序非递增、行数足够。
func parse2026File(path string, year int) (string, []*core.YiFenYiDuan, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	var f yfd2026File
	if err := json.Unmarshal(b, &f); err != nil {
		return "", nil, fmt.Errorf("解析 JSON：%w", err)
	}
	slug := f.Slug
	if slug == "" {
		slug = strings.TrimSuffix(filepath.Base(path), ".json")
	}
	p := mustProv(slug) // 未知 slug 直接退出（防错配）
	allowed := map[string]bool{}
	for _, t := range p.tracks {
		allowed[t] = true
	}

	var out []*core.YiFenYiDuan
	for _, tr := range f.Tracks {
		track := canonTrackName(tr.Track)
		if !allowed[track] {
			fmt.Fprintf(os.Stderr, "  · %s：科类 %q 不在 %s 的科类集 %v，跳过\n", slug, tr.Track, p.name, p.tracks)
			continue
		}
		if len(tr.Entries) == 0 {
			continue
		}
		y := &core.YiFenYiDuan{Province: p.name, Track: track, Year: year, ControlLine: tr.ControlLine, Entries: tr.Entries}
		core.SortFenduanAscending(y) // 分数升序（=> 累计降序）
		deriveCounts(y)              // 本段人数由累计派生（单一真相：累计是真值，count 仅展示且站点未用）
		if err := validateYFD(y); err != nil {
			fmt.Fprintf(os.Stderr, "  · %s %s：校验未过（%v），跳过该科类\n", slug, track, err)
			continue
		}
		out = append(out, y)
	}
	return slug, out, nil
}

// deriveCounts 用累计反推本段人数（Entries 须已按分数升序）。累计是真值、count 是冗余派生：
// count[i] = cum[i] − cum[i+1]（更低分 i 的本段人数 = 该分及其上 − 更高分及其上；中间缺列的分=0人，差值仍成立）；
// 最高分行 count = 其累计（其上无更高分行，得该分及以上者即累计）。
// 这样既消除来源里 count 列的不一致（如广西），又对本就自洽的省是恒等变换。
func deriveCounts(y *core.YiFenYiDuan) {
	n := len(y.Entries)
	for i := 0; i < n-1; i++ {
		y.Entries[i].Count = y.Entries[i].Cumulative - y.Entries[i+1].Cumulative
	}
	if n > 0 {
		y.Entries[n-1].Count = y.Entries[n-1].Cumulative
	}
}

// validateYFD 是入库前最后一道闸。Entries 已按分数升序（entries[0]=最低分）。
// 不变量：分数越低累计越大，故升序看累计应单调不增（cum[i] ≥ cum[i+1]）；累计全为正；行数过少视为不完整。
func validateYFD(y *core.YiFenYiDuan) error {
	if len(y.Entries) < 100 {
		return fmt.Errorf("仅 %d 行，疑似稀疏摘要而非完整表（完整表数百行）", len(y.Entries))
	}
	for i, e := range y.Entries {
		if e.Cumulative <= 0 {
			return fmt.Errorf("分 %d 累计 %d ≤ 0", e.Score, e.Cumulative)
		}
		if i+1 < len(y.Entries) && e.Cumulative < y.Entries[i+1].Cumulative {
			return fmt.Errorf("分 %d 累计 %d < 更高分 %d 的累计 %d（非单调，数据错）",
				e.Score, e.Cumulative, y.Entries[i+1].Score, y.Entries[i+1].Cumulative)
		}
	}
	return nil
}

// canonTrackName 把抓取里可能出现的「物理类/历史类/综合类」归一成站点科类名（物理/历史/综合）。
// 理科/文科（新疆老高考）原样保留。
func canonTrackName(s string) string {
	s = strings.TrimSpace(s)
	switch s {
	case "物理类":
		return "物理"
	case "历史类":
		return "历史"
	case "综合类":
		return "综合"
	}
	return s
}
