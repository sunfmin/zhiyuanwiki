package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

// locatorEntry 是一个可填报单位（黑龙江=组内专业；浙江=院校×专业），含挂接的往年/等效位次，
// 供客户端定位。JSON 键缩短以压小客户端体积。组相关字段（gc/gn/gs）仅黑龙江有。
type locatorEntry struct {
	SchoolCode string `json:"sc"`
	SchoolName string `json:"sn"`
	GroupCode  string `json:"gc,omitempty"`
	GroupName  string `json:"gn,omitempty"`
	MajorName  string `json:"mn"`
	MajorKey   string `json:"mk"`
	SelKe      string `json:"sk"`
	Plan       int    `json:"pl"`
	Rank       int    `json:"r"`            // 等效位次（无则往年最低位次）
	PrevYear   int    `json:"py"`           // 挂接到的年份
	GroupSize  int    `json:"gs,omitempty"` // 组内专业数（黑龙江服从调剂提示）
	Menlei     string `json:"mc,omitempty"` // 学科门类 1 字码（过滤用）
	Tuition    int    `json:"tu,omitempty"` // 学费（元/年，待定/无→0）
	Coop       bool   `json:"cw,omitempty"` // 中外合作办学
}

// dingweiCmd 从已生成的 school 详情构建按科类分片的定位索引到 public/data/<slug>/。
// 依赖先跑过 `yuanxiao -prov <slug>`。
func dingweiCmd(args []string) {
	fs := flag.NewFlagSet("dingwei", flag.ExitOnError)
	in := fs.String("in", filepath.Join("src", "data"), "src/data 目录（其下按省份 slug 分目录）")
	out := fs.String("out", filepath.Join("public", "data"), "定位索引输出目录（其下按省份 slug 分目录）")
	provSlug := fs.String("prov", "hlj", "省份 slug：hlj / zj")
	_ = fs.Parse(args)
	p := mustProv(*provSlug)

	schoolsDir := filepath.Join(srcDir(*in, p), "schools")
	files, err := filepath.Glob(filepath.Join(schoolsDir, "*.json"))
	if err != nil || len(files) == 0 {
		fatal(fmt.Errorf("未找到 %s 的 school 详情（先跑 yuanxiao -prov %s）：%v", p.name, p.slug, err))
	}

	byTrack := map[string][]locatorEntry{}
	for _, fp := range files {
		b, err := os.ReadFile(fp)
		if err != nil {
			fatal(err)
		}
		var d schoolDetail
		if err := json.Unmarshal(b, &d); err != nil {
			fatal(fmt.Errorf("%s: %w", fp, err))
		}
		// 黑龙江：院校专业组 → 组内专业
		for _, g := range d.Groups2026 {
			// 同组内同一专业键可能因计划侧子方向拆行而重复（如复旦工科试验班在源计划拆成 13 行子方向，
			// 归一后同名同 mk），合并为一条、计划求和——否则定位列同专业重复刷屏、撞 React key、组内专
			// 业数虚高。组内专业数（gs）按去重后计。根因在 core.BuildGroups2026 逐行建组，此处为视图层兜底。
			majors := dedupGroupMajors(g.Majors)
			size := len(majors)
			for _, m := range majors {
				if m.PrevRank <= 0 {
					continue // 无往年位次无法分档
				}
				rank := m.EquivRank
				if rank <= 0 {
					rank = m.PrevRank
				}
				byTrack[g.Track] = append(byTrack[g.Track], locatorEntry{
					SchoolCode: d.Code, SchoolName: d.Name,
					GroupCode: g.GroupCode, GroupName: g.GroupName,
					MajorName: m.MajorName, MajorKey: m.MajorKey,
					SelKe: m.SelKe, Plan: m.Plan,
					Rank: rank, PrevYear: m.PrevYear, GroupSize: size,
					Menlei: m.Menlei, Tuition: core.ParseTuition(m.Tuition), Coop: m.Coop,
				})
			}
		}
		// 专业平行志愿（无组）：院校×专业。浙江单科类（Track 空）落「综合」；双科类省（重庆/辽宁…）按行 Track 分片。
		for _, m := range d.Plan2026 {
			if m.PrevRank <= 0 {
				continue
			}
			rank := m.EquivRank
			if rank <= 0 {
				rank = m.PrevRank
			}
			track := m.Track
			if track == "" {
				track = zjTrack
			}
			byTrack[track] = append(byTrack[track], locatorEntry{
				SchoolCode: d.Code, SchoolName: d.Name,
				MajorName: m.MajorName, MajorKey: m.MajorKey,
				SelKe: m.SelKe, Plan: m.Plan,
				Rank: rank, PrevYear: m.PrevYear,
				Menlei: m.Menlei, Tuition: core.ParseTuition(m.Tuition), Coop: m.Coop,
			})
		}
	}

	outDir := pubDir(*out, p)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatal(err)
	}
	for track, entries := range byTrack {
		slug, ok := trackSlug[track]
		if !ok {
			continue
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Rank < entries[j].Rank })
		name := "locator-" + slug + ".json"
		writeJSON(filepath.Join(outDir, name), entries)
		fmt.Printf("✓ %s · %s类定位索引：%d 个可填报单位 → %s\n", p.name, track, len(entries), name)
	}
}

// zjTrack 是浙江唯一科类名（与 internal/zj.Track 一致），用于定位分片键。
const zjTrack = "综合"

// dedupGroupMajors 合并组内同 MajorKey 的专业（计划求和），保留首次出现的顺序与其余字段。
// 计划侧子方向拆行会让同一专业在组内重复多行（同名→同 mk），定位视图需归并为一条。
func dedupGroupMajors(majors []core.GroupMajor) []core.GroupMajor {
	idx := make(map[string]int, len(majors))
	out := make([]core.GroupMajor, 0, len(majors))
	for _, m := range majors {
		if i, ok := idx[m.MajorKey]; ok {
			out[i].Plan += m.Plan
			continue
		}
		idx[m.MajorKey] = len(out)
		out = append(out, m)
	}
	return out
}
