#!/usr/bin/env bash
# reimport-all.sh — 把全部省份的官方 xlsx 重新导入 SQLite staging 库（只做 import 步）。
#
# 稳妥策略（不直接在 iCloud 文件上跑 31 次写事务）：
#   1) 备份规范库 → <库>.bak-<时间戳>
#   2) seed 本地工作副本 out/zhiyuan.db（从规范库拷贝，保证失败省保留原数据）
#   3) 逐省 import 进 out/zhiyuan.db（首省刷新全国表，其余 -skip-national；按省幂等整省替换）
#   4) WAL checkpoint 后，把 out/zhiyuan.db 拷回规范库
#   5) 打印每省 pass/fail 与 plan 行数汇总
#
# 单省失败不中断整体（该省保留 seed 里的原数据），最后统一报告。
#
# 用法:
#   scripts/reimport-all.sh                       # 导入全部省份
#   scripts/reimport-all.sh js zj xz              # 只导入指定省份
#   ZHIYUAN_DB=/path/to/zhiyuan.db scripts/reimport-all.sh   # 覆盖规范库路径
set -uo pipefail
cd "$(dirname "$0")/.."

DB="${ZHIYUAN_DB:-/Users/sunfmin/Library/Mobile Documents/com~apple~CloudDocs/zhiyuanwiki/zhiyuan.db}"
GEFEN="${GEFEN:-$HOME/Downloads/高考志愿/各省份}"   # 多数省 xlsx 源根
DL="${DL:-$HOME/Downloads/高考志愿}"                  # 西藏独立包(31、西藏-…)的上层——已随数据归整挪入 高考志愿/
WORK="out/zhiyuan.db"                                # 本地工作副本
BIN="./zhiyuan-data"

# 全部支持的省份（provParsers ∪ {zj, shanxi}）。首个为「刷新全国表」省（用 各省份 源）。
ALL=(js hn sc ah gx sx bj sh hain nm gd fj nx jx jl gs cq gz ln hebei sd qh tj xj xz hb yn henan hlj zj shanxi)
PROVS=("$@"); [ "$#" -eq 0 ] && PROVS=("${ALL[@]}")

# 某省的 xlsx 源根：西藏用 ~/Downloads（provDirName 给包名），其余用 各省份。
src_for() { case "$1" in xz) echo "$DL";; *) echo "$GEFEN";; esac; }

[ -f "$DB" ] || { echo "✗ 规范库不存在: $DB" >&2; exit 1; }
[ -d "$GEFEN" ] || { echo "✗ xlsx 源根不存在: $GEFEN" >&2; exit 1; }

echo "▶ 构建 zhiyuan-data ..."
go build -o "$BIN" ./cmd/zhiyuan-data || { echo "✗ 构建失败" >&2; exit 1; }

TS="$(date +%Y%m%d-%H%M%S)"
echo "▶ 备份规范库 → ${DB}.bak-${TS}"
cp "$DB" "${DB}.bak-${TS}" || { echo "✗ 备份失败" >&2; exit 1; }

echo "▶ seed 本地工作副本 ${WORK}（从规范库）"
mkdir -p out
rm -f "$WORK" "$WORK-wal" "$WORK-shm"
cp "$DB" "$WORK" || { echo "✗ seed 失败" >&2; exit 1; }

declare -a OK=() FAIL=()
first=1
for slug in "${PROVS[@]}"; do
  src="$(src_for "$slug")"
  echo; echo "════════════════════ $slug (src=$src) ════════════════════"
  if [ "$first" -eq 1 ]; then
    "$BIN" import -prov "$slug" -src "$src" -db "$WORK"; rc=$?
    first=0
  else
    "$BIN" import -prov "$slug" -src "$src" -db "$WORK" -skip-national; rc=$?
  fi
  if [ "$rc" -eq 0 ]; then OK+=("$slug"); else FAIL+=("$slug"); echo "⚠ $slug 导入失败（保留原数据），继续"; fi
done

echo; echo "▶ WAL checkpoint"
sqlite3 "$WORK" "PRAGMA wal_checkpoint(TRUNCATE);" >/dev/null 2>&1 || true

echo "▶ 拷回规范库 $DB"
cp "$WORK" "$DB" || { echo "✗ 拷回失败——工作副本仍在 ${WORK}，备份在 ${DB}.bak-${TS}" >&2; exit 1; }

echo; echo "════════════════════ 汇总 ════════════════════"
echo "成功(${#OK[@]}): ${OK[*]:-无}"
echo "失败(${#FAIL[@]}): ${FAIL[*]:-无}"
echo; echo "各省 plan 行数（导入后）:"
sqlite3 -column "$DB" "SELECT prov, count(*) FROM plan GROUP BY prov ORDER BY prov;"
echo; echo "✓ 完成。备份：${DB}.bak-${TS}"
