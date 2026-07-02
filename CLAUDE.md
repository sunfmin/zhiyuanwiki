# zhiyuanwiki

## 改完流程：先测试，再开浏览器交人工

每次改完代码，**先用测试确认通过**，然后**打开浏览器交给我手动测试**——绿测试不等于完工。

1. 测试：逻辑改动跑 `npm test`（单元）；任何 UI 改动跑 `npm run build && npm run test:render`（渲染，慢）。必须全绿。
   - 渲染测试的机制：**先 `npm run build` 出 `dist/`，测试再用 `astro preview` serve 这份 `dist/`，Playwright 开真浏览器截图断言**（见 `tests/render/render-glue.ts` 的 `startPreview`）。
   - 故 `dist/` 是构建产物、非实时：**改完代码必须重新 `npm run build`，否则渲染测试跑的还是旧页面**。（dev server `npm run dev` 才是实时 HMR，那是给人工看的、不是渲染测试用的。）
2. 通过后起 **`astro preview`**（不是 `npm run dev`）并打开改动的页面——交人工测试统一用 preview serve 那份刚 build 的 `dist/`，跟渲染测试同源，所见即测过的：

   ```
   npm run build            # 若上一步已 build 过可略；preview serve 的是 dist/
   npm run preview          # 打印 URL，如 http://localhost:4321（端口被占会自增）
   open <打印出的 URL>/zj/   # 改了哪个页面就开哪个；macOS 用 open
   ```

   把要看的 URL/页面告诉我，并保持 server 运行。

## 数据管线（构建期）

构建期 SQLite staging 库（官方 xlsx → 结构化数据的中转库，见 ADR-0014）**规范位置**：

```
/Users/sunfmin/Library/Mobile Documents/com~apple~CloudDocs/zhiyuanwiki/zhiyuan.db
```

工具 `cmd/zhiyuan-data`（`go build -o zhiyuan-data ./cmd/zhiyuan-data`）各子命令默认 `-db out/zhiyuan.db`，
指向上面这份库时传 `-db "<上述路径>"`。原始 xlsx 与该库都是本机产物、不入仓库。

### xlsx 源目录（2026-07-01 全量审计）

数据规范根：`~/Downloads/高考志愿/`。选文件规则：在某省子树内按「路径含指定关键子串、体积最大者」挑，
`mustNot=艺术/艺考`；每省的关键子串在 `cmd/zhiyuan-data/import.go` 的 `provParsers`（`ScoreMust`/`PlanMust`）里登记。

| 数据 | 省 | 源根 |
|---|---|---|
| 分数/计划/一分一段 | 26 省（除下面特殊省） | `~/Downloads/高考志愿/各省份/<省>/…` |
| 全国院校属性 / 专业门类 | — | `各省份/…/college_data/全国高等院校{信息汇总,开设专业汇总}.xlsx`（遍历首个命中） |
| 西藏 `xz` | 分数/计划（无一分一段） | `高考志愿/31、西藏-2026志愿填报资料/…`（`import -prov xz -src ~/Downloads/高考志愿`） |
| 山西 `shanxi` | 计划/分数/一分一段 | `各省份/山西/…`（分数文件名标 2024、实为 2025 新高考，见 `import_shanxi.go`） |
| 黑龙江 `hlj` | 一分一段(2020-2025) | `各省份/黑龙江/…` |
| 黑龙江 `hlj` | 分数(23-25)/计划(26) | `高考志愿/24-万师兄-黑龙江2026年高考志愿填报大数据/…` |
| 黑龙江 `hlj` | 一分一段(2026·基准年) | `高考志愿/黑龙江2026物理类一分一段表.xlsx`（散在根，`各省份/黑龙江` 无 2026） |
| 浙江 `zj` | 分数/计划/院校属性 | `高考志愿/09、浙江-2026高考志愿填报资料/…` |

`zj`/`hlj` 的源根由 `defaultSrc()`=`~/Downloads/高考志愿` 给出（`import_zj.go`/`import_hlj.go`），通用省的
`各省份` 由它派生（`filepath.Join(defaultSrc(), "各省份")`）——**全部数据都在 `~/Downloads/高考志愿/` 一个根下**。
（历史：`defaultSrc` 原指万师兄树 `~/Developments/zhiyuan/官方数据`，2026-07-01 随数据归整收口到 `高考志愿/`。）

**溯源**：`import` 每选一个源文件都会打印 `📄 <用途> ← <绝对路径>`，任何时候都能核对实际用的是哪份（多版本同名时尤其有用）。

**重新导入全部省份 excel → db**（按省幂等整省替换，首省刷新全国表，其余 `-skip-national`）：
见 `scripts/reimport-all.sh`（先 seed 本地副本、逐省导入、成功后拷回规范位置，容错并汇总）。
**刷新站点 JSON**（不 import）：`scripts/refresh-json.sh`（每省 `fenduan→yuanxiao→zhuanye→dingwei`、末尾 `landing`，跳过无一分一段的西藏）。
### 源文件必须留存·绝不放临时目录

**所有从互联网获取的源数据（官方 PDF / 图片 JPG·PNG / 抓下来的 HTML / xlsx / 招生计划 / 万师兄大数据…，含一分一段），一律永久存到 `~/Downloads/高考志愿/`，绝不放 `/tmp`、scratchpad 等临时目录**——临时目录会被清掉，源文件丢了就得重扒官网、无法复现。源文件不入仓库，只提交 Go 管线生成的 JSON，但源文件本身必须在本机长期留着。

- 新扒的官方原件（尤其图片版 PDF/JPG/PNG 这类要 OCR 的）放该省下 `2026源/`（按年份）子目录，保留官方原文件名或加「-官方原件」后缀，便于回溯。
- 管线只认路径含「一分一段表」子串的 `.xlsx`；OCR 重塑出的规范表放 `.../一分一段表/<年>/<省>YYYY年的一分一段表.xlsx`，与 `2026源/` 原件分开存、互不干扰。

## Agent skills

### Issue tracker

Issues and PRDs are tracked as **GitHub issues** (via the `gh` CLI); external PRs are **not** a triage surface. See `docs/agents/issue-tracker.md`.

### Triage labels

Default 1:1 vocabulary — `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context layout — one `CONTEXT.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.
