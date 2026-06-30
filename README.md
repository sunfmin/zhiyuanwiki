# 高考志愿数据 Wiki（黑龙江 · 浙江）

用历年官方数据帮助高考考生选择院校与专业的公开 wiki。数据百科 + 方法论混合站，
Astro 静态生成 + 客户端位次定位 island，无服务端。对称 `/[省]/` 路由：`/hlj/`（黑龙江，
物理/历史）、`/zj/`（浙江，综合·专业平行志愿），根 `/` 跳默认省。

设计文档见 [`CONTEXT.md`](./CONTEXT.md)（术语表）与 [`docs/adr/`](./docs/adr/)（架构决策；
多省份泛化见 [`0009`](./docs/adr/0009-multi-province-zhejiang.md)）。

## 结构

- `src/` — Astro 站点（页面 `src/pages/[prov]/…`、布局、Preact island、方法论文章）
- `cmd/zhiyuan-data/` — Go 数据预处理工具：官方 xlsx → 结构化 JSON（`-prov` 选省份）
- `internal/core` — 省份无关原语；`internal/hlj`、`internal/zj` — 各省专属解析
- `src/data/<省>/`、`public/data/<省>/` — 预处理生成、提交进仓库的 JSON（原始 xlsx 不入仓库）

## 开发

前端（需要 Node ≥ 20）：

```sh
npm install
npm run dev      # 本地开发
npm run build    # 静态产出到 dist/
npm test         # vitest
```

数据工具（需要 Go ≥ 1.24）：

```sh
go build ./...
go test ./...
go run ./cmd/zhiyuan-data help
```

## 部署（Cloudflare R2 + Rules）

静态产出托管在 **Cloudflare R2**，前面用 **Cloudflare Rules**（非 Worker）服务于自定义域
`zhiyuanwiki.com`（ADR-0018，推翻 ADR-0006 的 Pages——免费版 2 万文件上限挡不住本站 4 万+ 页）。
CI 只跑 `npm run build`（读仓库内已提交的 `src/data` JSON）；**不在 CI 跑 Go 预处理**——原始
xlsx 不入仓库，数据由维护者本地 `go run ./cmd/zhiyuan-data ...` 生成后提交。

`.github/workflows/deploy.yml`：`npm run build` → `rclone sync dist → R2`（`--transfers=8 --retries 5`
避开 R2 高并发下的间歇性 501 NotImplemented）→ purge 缓存。回滚 = 对上个 commit 重跑 CI（构建对已提交 JSON 确定性可复现）。

**仓库 Secrets**：`R2_ACCESS_KEY_ID`、`R2_SECRET_ACCESS_KEY`（R2 的 S3 API 令牌）、
`CLOUDFLARE_ACCOUNT_ID`、`CLOUDFLARE_ZONE_ID`、`CLOUDFLARE_API_TOKEN`（需含 **Zone › Cache Purge** 权限）。

**Cloudflare 仪表盘一次性配置**：

1. 建 R2 桶 `zhiyuanwiki`（保持私有），**Connect Domain** 挂 `zhiyuanwiki.com`（apex；若原挂在 Pages 先 detach）。
   R2 自定义域**原生服务目录 index.html**：`/zj/` 取 `zj/index.html`，`/zj/index.html` 与 `/zj` 均 308 收敛到 `/zj/`。
   故构建用 `trailingSlash:'always'`（目录式产物 + 站内链接带尾斜杠匹配 canonical，由
   `tests/render/internal-links.trailing-slash.test.ts` 守护）。
   ⚠ **不要**再加「补 index.html」的 Transform/Redirect 规则——会与 R2 的 clean-URL 308 互相打架成无限重定向，见 ADR-0019。
2. **Cache Rule**：让 `text/html` 进边缘缓存（长 edge TTL）；每次部署末尾 CI 自动 `purge_everything`。

> 不需要 URL Rewrite 或 Bulk Redirects：R2 原生处理目录索引；旧单省 URL → 黑龙江 的边缘重定向（ADR-0009）已退役（`public/_redirects` 在 R2 下不生效，已删）。

## 重新生成数据

每个命令用 `-prov hlj|zj` 选省份（默认 hlj），产物落到 `src/data/<省>/`、`public/data/<省>/`：

```sh
for P in hlj zj; do
  go run ./cmd/zhiyuan-data fenduan  -prov $P   # 一分一段 → JSON
  go run ./cmd/zhiyuan-data yuanxiao -prov $P   # 院校 / 院校×专业 / 2026 报考视图
  go run ./cmd/zhiyuan-data zhuanye  -prov $P   # 专业跨校聚合
  go run ./cmd/zhiyuan-data dingwei  -prov $P   # 位次定位索引
done
```

渲染测试（Playwright，前置 `npm run build`）：`npm run test:render`。

## 数据来源与免责

数据源自各省招生考试院官方公开数据（黑龙江 lzk.hl.cn；浙江省教育考试院），及万师兄·
高考志愿填报大数据（第三方整理）。本站数据**仅供参考**，一切以各省考试院及院校最新官方公布为准。
