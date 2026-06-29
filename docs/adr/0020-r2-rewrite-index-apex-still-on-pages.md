# 服务层订正：R2 经「自定义域 + URL Rewrite 补 index.html」服务；apex 仍在 Pages（推翻 ADR-0019 的「R2 原生索引」误判）

ADR-0019 断言 apex `zhiyuanwiki.com` 已在 R2，且「R2 自定义域**原生**服务目录索引，并把 `…/index.html` 308 收敛到 `…/`」，据此**删掉**了补 index.html 的 URL Rewrite，并立下「**永不再加**该规则」的护栏。这两个前提经实测均不成立。

## 订正（实测推翻 0019 的前提）

1. **apex 当前由 Cloudflare Pages 托管，不是 R2。** Pages 项目 `zhiyuanwiki` 持有域名 `zhiyuanwiki.com` 与 `zhiyuanwiki.pages.dev`；二者行为完全一致（目录索引、`…/index.html → …/` 的 308、且含新省份 `xj`）。**目录索引与 clean-URL 308 是 Pages 的原生行为。** wrangler 亦确认：R2 桶上**无任何自定义域**。

2. **R2 不解析目录 index.html——自定义域亦然**（Cloudflare 官方明示：需 Worker 或 Rules）。实测裸 R2 `pub-…r2.dev`：`/`、`/zj/yuanxiao/2219/` 皆 **404**，仅精确 key `…/index.html` 才 200。

3. **2026-06-29 的 `ERR_TOO_MANY_REDIRECTS` 是 Pages 的 clean-URL 308 与补 index.html Rewrite 打架，与 R2 无关。** 决定性证据：在**纯 R2** 自定义域 `r2.zhiyuanwiki.com` 上，`/zj/yuanxiao/2219/index.html` → **200（无 308）**，故 Rewrite + 纯 R2 **不成环**。ADR-0019 把 Pages 的 308 误读成了 R2 行为（当时 apex 实为 Pages）。

## 决定

1. **R2 经「自定义域 + 一条 URL Rewrite」服务**（zone `http_request_transform` 阶段）：
   - 条件：`(http.host eq "r2.zhiyuanwiki.com" and ends_with(http.request.uri.path, "/"))`
   - 动作：rewrite path → `concat(http.request.uri.path, "index.html")`
   - **host-scoped 到 `r2.zhiyuanwiki.com`**（纯 R2 域），apex（Pages）不受影响、不成环。

2. **选 Rewrite Rule 而非 Worker**：
   - Worker 免费版 **10 万次/天硬顶**，且**每个请求都计**——Worker 跑在 Cloudflare 缓存**之前**，缓存命中也计数；超限回 **Error 1027** 宕站（高考季峰值有真实风险）。付费 $5/月起。
   - URL Rewrite 是 CDN 内建、**无请求次数上限**、免费。
   - R2 又**无文件数上限**；对照：Pages 2 万顶、Workers Static Assets 免费 2 万 / 付费 10 万顶——本站 ~4 万 HTML，唯 R2 永久无忧。

3. **沿用 `trailingSlash:'always'`** 与 ADR-0018 已接受的两项代价：无斜杠 `/zj` → R2 404（站内链接恒带斜杠，恒命中 Rewrite）；404 为 R2 默认页（较丑）。

4. **废除 ADR-0019「永不再加补 index.html 规则」的护栏**——其依据（R2 自带 308）系误判。新护栏：**该 Rewrite 必须 host-scoped 到纯 R2 域，切勿施于 Pages 服务的 host**（否则复现死循环）。

## 实测验证（`r2.zhiyuanwiki.com`，规则上线后）

| 请求 | 结果 |
| --- | --- |
| `/` | `200`（Rewrite → `/index.html`） |
| `/zj/yuanxiao/2219/` | `200` |
| `/zj/yuanxiao/2219/index.html` | `200`（无 308，**不成环**） |
| `/zj/yuanxiao/2219`（无斜杠） | `404`（既定代价） |
| `/favicon.svg`、`/xj/` | `200` |

## 现状与后续

- `r2.zhiyuanwiki.com` 已正确服务 R2 全量（内容仍由部署管线 `rclone sync dist → 桶` 更新；见 deploy.yml）。
- **apex `zhiyuanwiki.com` 仍在 Pages**（最后部署偏旧，且 Pages 2 万文件顶仍在——迁移的初衷未真正完成）。
- **后续（未做，待定）切 apex 到 R2**：把自定义域 `zhiyuanwiki.com` 从 Pages 改挂 R2 桶，并把上面 Rewrite 的 host 条件改/加到 apex，退役 Pages 项目。务必**先确认 apex 已不在 Pages（无 Pages 的 308），再加 Rewrite**。

## 与既有 ADR 的关系

- **订正 ADR-0019** 的核心前提（apex 在 R2、R2 原生索引、R2 自带 clean-URL 308）及其「永不补 index.html」护栏。
- **附带订正 ADR-0018** 部署节「R2 不支持 CRC32 → 501」一段：实测该 501 来自 rclone 上传后的校验 `HEAD …?versionId=`（R2 未实现 versionId），已在 deploy.yml 用 `--s3-no-head` 根治，与 CRC32 无关。
- R2 托管、rclone 增量同步、缓存 Purge、`trailingSlash:'always'` 仍有效。

## 更新（2026-06-29，apex 已切到 R2）

apex `zhiyuanwiki.com` 的 cutover 已完成：

- 从 Pages 项目移除该自定义域（dashboard）；其 R2 自定义域绑定本就 ssl-active（apex 曾**双挂** Pages + R2，Pages 在边缘抢先），释放后即由 R2 服务。
- 把上面 Rewrite 的 host 条件扩到 `(http.host eq "zhiyuanwiki.com" or http.host eq "r2.zhiyuanwiki.com")`——**务必在 Pages 释放之后**才加 apex，否则复现死循环。
- 整 zone `purge_everything` 清掉 Pages 残留缓存；apex 自定义域 minTLS `1.0 → 1.2`。
- 实测（cache-busted，避开缓存）：apex `/`、`/zj/`、`/xj/`、`/gd/yuanxiao/10753/`、深层院校页（75 KB）、`/data/hn/locator-lishi.json`（1.7 MB）全 `200`，首页 `<title>` 正常；`/zj/yuanxiao/2219`（无斜杠）`404`（既定代价）；`/…/index.html` `200` 不成环。

**Pages 项目保留**（仅摘掉 apex 自定义域，`zhiyuanwiki.pages.dev` 仍在）作回滚兜底：R2 若出问题，把 apex 自定义域重新加回 Pages 即回退。彻底退役可后续 `wrangler pages project delete zhiyuanwiki`。至此 ADR-0018 迁离 Pages 的初衷真正完成（apex 不再受 Pages 2 万文件顶约束）。
