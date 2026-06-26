package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// locatorEntry 是一个可填报的 2026 组内专业（含挂接的往年/等效位次），供客户端定位。
// JSON 键缩短以压小客户端体积。
type locatorEntry struct {
	SchoolCode string `json:"sc"`
	SchoolName string `json:"sn"`
	GroupCode  string `json:"gc"`
	GroupName  string `json:"gn"`
	MajorName  string `json:"mn"`
	MajorKey   string `json:"mk"`
	SelKe      string `json:"sk"`
	Plan       int    `json:"pl"`
	Rank       int    `json:"r"`  // 等效位次（无则往年最低位次）
	PrevYear   int    `json:"py"` // 挂接到的年份
	GroupSize  int    `json:"gs"` // 组内专业数（服从调剂提示）
}

// dingweiCmd 从已生成的 school 详情构建按科类分片的定位索引到 public/data/。
// 依赖先跑过 `yuanxiao`。
func dingweiCmd(args []string) {
	fs := flag.NewFlagSet("dingwei", flag.ExitOnError)
	in := fs.String("in", filepath.Join("src", "data", "schools"), "school 详情 JSON 目录")
	out := fs.String("out", filepath.Join("public", "data"), "定位索引输出目录")
	_ = fs.Parse(args)

	files, err := filepath.Glob(filepath.Join(*in, "*.json"))
	if err != nil || len(files) == 0 {
		fatal(fmt.Errorf("未找到 school 详情（先跑 yuanxiao）：%v", err))
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
		for _, g := range d.Groups2026 {
			size := len(g.Majors)
			for _, m := range g.Majors {
				if m.PrevRank <= 0 { // 没有往年位次的暂不进定位（无法分档）
					continue
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
				})
			}
		}
	}

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fatal(err)
	}
	trackFile := map[string]string{"物理": "locator-wuli.json", "历史": "locator-lishi.json"}
	for track, entries := range byTrack {
		sort.Slice(entries, func(i, j int) bool { return entries[i].Rank < entries[j].Rank })
		name := trackFile[track]
		if name == "" {
			continue
		}
		writeJSON(filepath.Join(*out, name), entries)
		fmt.Printf("✓ %s类定位索引：%d 个可填报组内专业 → %s\n", track, len(entries), name)
	}
}
