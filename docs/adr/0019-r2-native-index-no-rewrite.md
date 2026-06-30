# 索引交还 R2：移除「补 index.html」的 URL Rewrite（修订 ADR-0018 的服务层）

ADR-0018 迁托管到 R2 + Cloudflare Rules 时，**假设「R2 自定义域不服务目录 index.html」**，于是加了一条 Transform Rule（URL Rewrite）把目录式 URL 补全：

```
当 ends_with(http.request.uri.path, "/")  →  重写 path 为 concat(path, "index.html")
```

并以 `trailingSlash:'always'` 令站内链接全带尾斜杠、恒命中该重写。

## 事故（2026-06-29，全站宕机）

线上 `zhiyuanwiki.com` 全站 `ERR_TOO_MANY_REDIRECTS`。逐 URL 抓包：

| 请求 | 响应 |
| --- | --- |
| `/` | `308 → /`（自指循环） |
| `/index.html` | `308 → /` |
| `/zj/` | `308 → /zj/`（自指循环） |
| `/zj/index.html` | `308 → /zj/`（**此请求不以 `/` 结尾，Rewrite 不触发**，仍 308） |
| `/favicon.svg`、`/robots.txt`、`/_astro/*.css` | `200`（静态资源正常） |

关键判据：`/zj/index.html` 不触发 Rewrite 却仍 `308 → /zj/`，证明这个 308 来自 **R2 本身**——R2 自定义域**确实原生服务目录索引**，并把显式的 `…/index.html` 308 重定向到 clean URL `…/`。于是与 Rewrite 死锁：

```
/zj/  --[Transform 补 index.html]-->  /zj/index.html  --[R2 clean-URL 308]-->  /zj/  --> ∞
```

只有 HTML/目录页全挂；静态资源（路径不以 `/` 结尾、Rewrite 不触发）不受影响，故"资源能下、页面打不开"。

## 决定

1. **删除该 Transform Rule**，目录索引交还 R2 原生处理。验证（删除后）：`/`、`/zj/`、`/zj/yuanxiao/` 均 `200` 零跳；`/zj/index.html` `308 → /zj/` 单跳收敛；静态资源 `200`。
2. **保留 `trailingSlash:'always'`**，但理由变了：不再是「为命中 Rewrite」，而是
   - (a) Astro 须产出**目录式** `dist/zj/index.html`，R2 才有 `zj/index.html` 这个 key 可服务 `/zj/`（若 `'never'` 产出 `dist/zj.html`，R2 请求 `/zj/` 404）；
   - (b) 站内链接带尾斜杠以匹配 R2 canonical（`…/`），省去每次导航一跳 308。由 `tests/render/internal-links.trailing-slash.test.ts` 守护（该测试的不变量仍成立，仅语义从「防死链」变为「防多余 308 跳」）。
3. **仍不写 Worker**：R2 原生索引已覆盖 ADR-0018 当初想用 Rewrite 解决的全部场景。

## 护栏

- **永远不要再加「给目录 URL 补 index.html」的 Transform 或 Redirect 规则**——R2 会把 `…/index.html` 又 308 回 `…/`，必成无限重定向。`astro.config.mjs`、README 部署节、上方表格三处已留警示。
- zone 内规则现状（已用全权 token 核实）：`http_request_transform` 现为空；**无** `http_request_dynamic_redirect`（Redirect Rules）、**无** `http_request_redirect`（Bulk Redirects）entrypoint，无 Page Rules。唯一边缘逻辑就是 R2 自定义域自身 + Cache Rule。

## 顺带退役：旧单省 URL 重定向（ADR-0009）

ADR-0009 的 `public/_redirects`（8 条旧单省 URL → 黑龙江 `/hlj/...`）经核**在 R2 下从未生效**：R2 不读 `_redirects`，且 zone 内并无对应 Bulk Redirect。已删除仓库内失效的 `public/_redirects`，该重定向就此退役（不再在边缘承接旧 URL；旧链接将走 R2 404）。如确需保留，应在 Cloudflare Bulk Redirects 重建——本次决定不重建。

## 修订关系

本 ADR **修订** ADR-0018 的「服务层 = URL Rewrite 补 index.html」一条，并**退役** ADR-0009 的边缘重定向。ADR-0018 的其余决定（R2 托管、rclone 增量同步部署、git 回滚、Cache Rule + 部署末 purge）不变。
