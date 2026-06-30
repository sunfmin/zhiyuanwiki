package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

// ascTable 造一张合法的升序一分一段：分数 [lo,hi]，累计随分数升高而降（顶段=1）。
func ascTable(lo, hi int) []core.FenduanEntry {
	var es []core.FenduanEntry
	for s := lo; s <= hi; s++ {
		es = append(es, core.FenduanEntry{Score: s, Count: 1, Cumulative: hi - s + 1})
	}
	return es // 已按分数升序
}

func TestValidateYFD(t *testing.T) {
	good := ascTable(200, 700) // 501 行，单调
	nonMono := ascTable(200, 700)
	nonMono[100].Cumulative = 0 // 中间塞个 0：既非正、又破坏单调
	tests := []struct {
		name    string
		entries []core.FenduanEntry
		wantErr bool
	}{
		{"完整单调表通过", good, false},
		{"稀疏摘要被拒", ascTable(660, 700), true}, // 41 行 < 100
		{"非正累计被拒", nonMono, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			y := &core.YiFenYiDuan{Province: "江苏", Track: "物理", Year: 2026, Entries: tt.entries}
			err := validateYFD(y)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateYFD err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateYFDNonMonotonic(t *testing.T) {
	es := ascTable(200, 700)
	// 让某低分行的累计比相邻高分行还小 → 破坏「分越低累计越大」。
	es[300].Cumulative = es[301].Cumulative - 5
	y := &core.YiFenYiDuan{Province: "江苏", Track: "物理", Year: 2026, Entries: es}
	if err := validateYFD(y); err == nil {
		t.Fatal("非单调表应被拒，却通过了")
	}
}

func TestCanonTrackName(t *testing.T) {
	cases := map[string]string{
		"物理类": "物理", "历史类": "历史", "综合类": "综合",
		"物理": "物理", "理科": "理科", "文科": "文科", " 综合 ": "综合",
	}
	for in, want := range cases {
		if got := canonTrackName(in); got != want {
			t.Errorf("canonTrackName(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestDeriveCounts(t *testing.T) {
	// 升序：分 100(cum 50) / 101(cum 30) / 102(cum 12)。乱填 count，派生后应自洽。
	y := &core.YiFenYiDuan{Entries: []core.FenduanEntry{
		{Score: 100, Count: 999, Cumulative: 50},
		{Score: 101, Count: 0, Cumulative: 30},
		{Score: 102, Count: 7, Cumulative: 12},
	}}
	deriveCounts(y)
	want := []int{50 - 30, 30 - 12, 12} // 20, 18, 12（顶段 count=累计）
	for i, w := range want {
		if y.Entries[i].Count != w {
			t.Errorf("entry[%d].Count=%d, want %d", i, y.Entries[i].Count, w)
		}
	}
}

func TestParse2026File(t *testing.T) {
	// 江苏：物理(合法) + 历史(稀疏，应被丢) + 化学(不在科类集，应被丢)。
	f := yfd2026File{Slug: "js", Province: "江苏", Status: "ok"}
	f.Tracks = []struct {
		Track       string              `json:"track"`
		ControlLine int                 `json:"controlLine"`
		Entries     []core.FenduanEntry `json:"entries"`
	}{
		{Track: "物理类", ControlLine: 456, Entries: ascTable(300, 700)},
		{Track: "历史", ControlLine: 484, Entries: ascTable(680, 700)}, // 稀疏
		{Track: "化学", Entries: ascTable(300, 700)},                    // 非法科类
	}
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "js.json")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}

	slug, yfds, err := parse2026File(path, 2026)
	if err != nil {
		t.Fatalf("parse2026File: %v", err)
	}
	if slug != "js" {
		t.Errorf("slug=%q, want js", slug)
	}
	if len(yfds) != 1 {
		t.Fatalf("应只剩 1 个合法科类（物理），got %d", len(yfds))
	}
	y := yfds[0]
	if y.Track != "物理" {
		t.Errorf("科类归一应为「物理」，got %q", y.Track)
	}
	if y.Province != "江苏" || y.Year != 2026 || y.ControlLine != 456 {
		t.Errorf("省/年/控制线错：%+v", *y)
	}
	// 升序排稳：entries[0] 最低分。
	if y.Entries[0].Score != 300 {
		t.Errorf("应按分数升序，entries[0].Score=%d want 300", y.Entries[0].Score)
	}
}
