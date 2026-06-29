# 托管：迁移到 R2 + Cloudflare Rules（推翻 ADR-0006 的 Pages）

ADR-0006 选 Cloudflare Pages 托管静态产物。但 **Pages 免费版封顶「2 万文件/部署」**，而本站已 **4 万+ 文件**（31 省全量 × 院校 `yuanxiao` 2.0万 + 专业 `zhuanye` 1.9万 静态预渲染）。自 8 省批量接入起，每次 `pages deploy` 都失败：`Error: Pages only supports up to 20,000 files in a deployment for your current plan.`——**线上站已停在旧批次、长期陈旧**。

解锁 10 万文件上限需 **Cloudflare Pro（$20/月）**；已验证 `PAGES_WRANGLER_MAJOR_VERSION=4` + Workers Paid（$5/月）**不生效**（错误信息按 plan 判定，社区多例佐证）。$240/年不划算。

## 决定

迁到 **R2 对象存储 + Cloudflare Rules（不写 Worker）**，挂在既有自定义域名 **`zhiyuanwiki.com`（apex）** 前。R2 免费层足够：产物 1.4 GB / 10 GB，全量同步 ~4 万次 Class A / 100 万免费，**出口流量免费**。公网 URL 不变（apex 由 Pages 自定义域改挂 R2）。

1. **服务层 = Rules-only**。一条 URL Rewrite 把目录式 URL 补 `index.html`：`ends_with(http.request.uri.path, "/")` → `concat(path, "index.html")`（无正则，覆盖 `/` 与 `/zj/`；资源路径不以 `/` 结尾，原样透传）。**有意接受两项代价**：(a) 无斜杠的 `/zj` 命中 R2 404；(b) 404 是 R2 原始页面，比 Pages 默认 404 丑。
   - 为消化 (a)，**强制 `trailingSlash: 'always'`**（`astro.config.mjs`），令站内链接全带斜杠、重写规则恒命中。
   - 选 Rules 而非 Worker：换「不维护代码」，代价是上面两项 + 放弃 clean URL（无斜杠跳转需 `regex_replace`，属 Business $200/月，不走）。
2. **旧 URL 重定向**：`public/_redirects`（8 条旧单省 URL → `/hlj/...`，见 ADR-0009）迁到 **Cloudflare Bulk Redirects**（非 Single Redirects——免费版后者约 10 条上限；Bulk 容量大且支持通配）。`public/_redirects` 在 R2 下失效，删除。
3. **部署 = rclone 原地增量同步 + 删除护栏 + git 回滚**。CI 用 `rclone sync dist → bucket`（`wrangler r2 object put` 单对象、4 万次不可行）。**护栏**：新构建文件数比线上少 >5% 则中止（防半截构建 + `--delete` 误删全站）。**放弃 Pages 的即时回滚与不可变历史**：构建对已提交 JSON 确定性可复现（CI 不跑 Go，见 ADR-0005），回滚 = 对上个 commit 重跑 CI（~10 分钟，与维护者触发的低频部署相称）。未来若需即时回滚，再上「版本前缀 + 指针切换」。
4. **缓存 = 边缘缓存 HTML + 每次部署 Purge Everything**。Cache Rule 让 `text/html` 进边缘缓存（长 edge TTL）；部署末尾调 CF API `purge_everything`（既有 API token 需加 Cache Purge 权限）。整 zone 清，但 zone 专属本站，无碍。R2 读降到近零（仅 miss 回源），延迟最优，部署在 purge 返回即生效。

## 代价与边界

- **失去 Pages 白送的能力**：不可变部署、即时回滚、PR 预览分支、`_headers`/`_redirects`、Git 集成自动构建——改为自管 CI 同步 + 上述护栏/回滚替代。
- **R2 必须配自定义域**：`*.pages.dev` 是 Pages 专属、`*.r2.dev` 限流不可作生产，故 R2-as-website 强依赖 `zhiyuanwiki.com` 这个 zone（亦是 Rules 能在桶前运行的前提）。
- **桶建议保持私有 + 仅挂自定义域**（不开 `r2.dev` 公共访问）。

## 考虑过的备选

- **Cloudflare Pro $20/月** —— 留在 Pages、保住全部白送能力。否决：成本不划算（$240/年），且不解决「文件数随数据增长」的根因。
- **R2 + Worker（~30 行）** —— 可一并解决 index 解析、旧重定向、clean URL、友好 404，桶保持私有。否决：选择 Rules-only 以「零代码维护」，接受上述两项代价。
- **版本前缀 + 指针切换的原子部署** —— 即时回滚、无误删风险。推迟：当前在地同步 + 护栏 + git 回滚已够，待真正需要即时回滚再上。
