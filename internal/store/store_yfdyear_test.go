package store

import (
	"testing"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

func yfd(year int, track string, n int) *core.YiFenYiDuan {
	y := &core.YiFenYiDuan{Province: "江苏", Track: track, Year: year, ControlLine: 456}
	for i := 0; i < n; i++ {
		y.Entries = append(y.Entries, core.FenduanEntry{Score: 700 - i, Count: 1, Cumulative: i + 1})
	}
	return y
}

// countYear 数某省某年入库的行数（跨科类）。
func countYear(t *testing.T, db *DB, prov string, year int) int {
	t.Helper()
	yds, err := db.LoadYiFenYiDuan(prov, "江苏")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, y := range yds {
		if y.Year == year {
			n += len(y.Entries)
		}
	}
	return n
}

func TestReplaceYiFenYiDuanYearAdditive(t *testing.T) {
	db := openTmp(t)
	// 先按通用 import 写入 2025（物理5 + 历史3）。
	if err := db.ReplaceYiFenYiDuan("js", []*core.YiFenYiDuan{yfd(2025, "物理", 5), yfd(2025, "历史", 3)}); err != nil {
		t.Fatal(err)
	}
	// additive 补 2026 物理(4)，不应动 2025。
	if err := db.ReplaceYiFenYiDuanYear("js", 2026, []*core.YiFenYiDuan{yfd(2026, "物理", 4)}); err != nil {
		t.Fatal(err)
	}
	if got := countYear(t, db, "js", 2025); got != 8 {
		t.Errorf("2025 应保持 8 行，got %d", got)
	}
	if got := countYear(t, db, "js", 2026); got != 4 {
		t.Errorf("2026 应 4 行，got %d", got)
	}
	// 幂等重导 2026（换成 6 行）：只重写 2026，2025 仍不动。
	if err := db.ReplaceYiFenYiDuanYear("js", 2026, []*core.YiFenYiDuan{yfd(2026, "物理", 6)}); err != nil {
		t.Fatal(err)
	}
	if got := countYear(t, db, "js", 2026); got != 6 {
		t.Errorf("重导后 2026 应 6 行，got %d", got)
	}
	if got := countYear(t, db, "js", 2025); got != 8 {
		t.Errorf("重导 2026 后 2025 仍应 8 行，got %d", got)
	}
}

func TestReplaceYiFenYiDuanYearRejectsYearMismatch(t *testing.T) {
	db := openTmp(t)
	// 传进来的 YFD 年份与目标年份不符 → 报错，防错配。
	if err := db.ReplaceYiFenYiDuanYear("js", 2026, []*core.YiFenYiDuan{yfd(2025, "物理", 4)}); err == nil {
		t.Fatal("年份不符应报错，却通过了")
	}
}
