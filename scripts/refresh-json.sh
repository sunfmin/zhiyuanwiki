#!/usr/bin/env bash
# refresh-json.sh — 从 SQLite staging 库投影全部省份的站点 JSON（只做投影，不 import）。
#
# 每省顺跑：fenduan（仅当该省有一分一段）→ yuanxiao → zhuanye → dingwei；全部省份后跑一次 landing。
# 产物（提交物）：src/data/<slug>/*  public/data/<slug>/*  src/data/{home-schools,benke-plan}.json
#
# DB 默认用本地工作副本 out/zhiyuan.db（reimport-all.sh 产出，内容=规范库）；
# 用 ZHIYUAN_DB=<路径> 可改指规范库。fenduan/yuanxiao/landing 读 DB；zhuanye/dingwei 读上一步 JSON。
#
# 用法:
#   scripts/refresh-json.sh                 # 全部省份
#   scripts/refresh-json.sh shanxi zj       # 只投影指定省份（landing 仍会跑，派生全国数据）
#   ZHIYUAN_DB=/path/to/zhiyuan.db scripts/refresh-json.sh
set -uo pipefail
cd "$(dirname "$0")/.."

DB="${ZHIYUAN_DB:-out/zhiyuan.db}"
BIN="./zhiyuan-data"
# 全部支持的省份（与 reimport-all.sh 一致）
ALL=(js hn sc ah gx sx bj sh hain nm gd fj nx jx jl gs cq gz ln hebei sd qh tj xj xz hb yn henan hlj zj shanxi)
PROVS=("$@"); [ "$#" -eq 0 ] && PROVS=("${ALL[@]}")

[ -f "$DB" ] || { echo "✗ 库不存在: $DB（先跑 reimport-all.sh，或用 ZHIYUAN_DB 指定）" >&2; exit 1; }

echo "▶ 构建 zhiyuan-data ..."
go build -o "$BIN" ./cmd/zhiyuan-data || { echo "✗ 构建失败" >&2; exit 1; }

# 该省一分一段段数；=0（如西藏「只有分数」省）则跳过 fenduan——否则 fenduanCmd 会 fatal。
yfd_count() { sqlite3 "$DB" "SELECT count(*) FROM yifenyiduan WHERE prov='$1';" 2>/dev/null || echo 0; }

declare -a OK=() FAIL=()
for slug in "${PROVS[@]}"; do
  echo; echo "════════════════════ $slug ════════════════════"
  ok=1
  if [ "$(yfd_count "$slug")" -gt 0 ]; then
    "$BIN" fenduan -prov "$slug" -db "$DB" || ok=0
  else
    echo "（$slug 无一分一段，跳过 fenduan）"
  fi
  "$BIN" yuanxiao -prov "$slug" -db "$DB" || ok=0
  "$BIN" zhuanye  -prov "$slug"           || ok=0
  "$BIN" dingwei  -prov "$slug"           || ok=0
  if [ "$ok" -eq 1 ]; then OK+=("$slug"); else FAIL+=("$slug"); echo "⚠ $slug 有步骤失败"; fi
done

echo; echo "════════════════════ landing（全国派生）════════════════════"
"$BIN" landing -db "$DB" || echo "⚠ landing 失败"

echo; echo "════════════════════ 汇总 ════════════════════"
echo "成功(${#OK[@]}): ${OK[*]:-无}"
echo "失败(${#FAIL[@]}): ${FAIL[*]:-无}"
echo "✓ 完成。改完记得 npm run build && npm run test:render 后人工核对。"
