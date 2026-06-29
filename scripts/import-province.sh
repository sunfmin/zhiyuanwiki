#!/usr/bin/env bash
# import-province.sh — 把一个或多个省份从官方 xlsx 走通整条构建期管线（ADR-0014）。
#
# 用法:
#   scripts/import-province.sh <slug>...        # 例: scripts/import-province.sh gx jx hb yn henan
#   SRC=/path/to/各省份 scripts/import-province.sh gx
#
# 每省按序跑 import → fenduan → yuanxiao → zhuanye → dingwei，全部省份跑完后跑一次 landing
# （全国级派生：本省院校数 + 各省本科招生计划总数）。
#
# 特性（稳妥）:
#   - 按省幂等：ReplaceXxx 整省替换，可反复重跑；out/zhiyuan.db 累积、不入仓库。
#   - 全国表（院校属性/专业门类）只在第一省刷新，其余省 -skip-national 省时。
#   - set -euo pipefail：任一步出错即停，不会把半截产物留给后续步骤。
#
# 产物（提交物）:
#   src/data/<slug>/*  public/data/<slug>/*  src/data/{home-schools,benke-plan}.json
# 原始 xlsx 与 out/zhiyuan.db 都是本机构建产物，不入仓库。
set -euo pipefail

cd "$(dirname "$0")/.."

if [ "$#" -eq 0 ]; then
  echo "用法: scripts/import-province.sh <slug>...   例: gx jx hb yn henan" >&2
  exit 2
fi

SRC="${SRC:-$HOME/Downloads/高考志愿/各省份}"
if [ ! -d "$SRC" ]; then
  echo "✗ 数据源目录不存在: $SRC（用 SRC=... 覆盖）" >&2
  exit 1
fi

echo "▶ 构建 zhiyuan-data ..."
go build -o zhiyuan-data ./cmd/zhiyuan-data

BIN=./zhiyuan-data
first=1
for slug in "$@"; do
  echo
  echo "════════════════════ $slug ════════════════════"
  if [ "$first" -eq 1 ]; then
    "$BIN" import -prov "$slug" -src "$SRC"      # 首省刷新全国表
    first=0
  else
    "$BIN" import -prov "$slug" -src "$SRC" -skip-national
  fi
  "$BIN" fenduan  -prov "$slug"
  "$BIN" yuanxiao -prov "$slug"
  "$BIN" zhuanye  -prov "$slug"
  "$BIN" dingwei  -prov "$slug"
done

echo
echo "════════════════════ landing（全国派生）════════════════════"
"$BIN" landing

echo
echo "✓ 完成：$* — 记得 npm run build && npm run test:render 后人工核对"
