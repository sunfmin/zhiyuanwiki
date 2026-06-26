# 高考志愿数据 Wiki（黑龙江先行）

用历年官方数据帮助高考考生选择院校与专业的公开 wiki。数据百科 + 方法论混合站，
Astro 静态生成 + 客户端位次定位 island，无服务端。

设计文档见 [`CONTEXT.md`](./CONTEXT.md)（术语表）与 [`docs/adr/`](./docs/adr/)（架构决策）。

## 结构

- `src/` — Astro 站点（页面、布局、Preact island、方法论文章）
- `cmd/zhiyuan-data/` — Go 数据预处理工具：官方 xlsx → 结构化 JSON
- `internal/` — Go 领域逻辑（位次换算、等效位次、挂接、选科判定）
- `src/data/` — 预处理生成、提交进仓库的 JSON（原始 xlsx 不入仓库）

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

## 部署（Cloudflare Pages）

静态产出托管在 Cloudflare Pages（ADR-0006）。CI 只跑 `npm run build`（读仓库内已提交的
`src/data` JSON）；**不在 CI 跑 Go 预处理**——原始 xlsx 不入仓库，数据由维护者本地
`go run ./cmd/zhiyuan-data ...` 生成后提交。

两种接法，二选一：

1. **Dashboard 连仓库**（推荐）：在 Cloudflare Pages 新建项目连本仓库，构建命令
   `npm run build`，输出目录 `dist`。之后推送 main 自动构建部署。
2. **GitHub Actions**：`.github/workflows/deploy.yml` 已就绪。在仓库 Secrets 配
   `CLOUDFLARE_API_TOKEN` 与 `CLOUDFLARE_ACCOUNT_ID`，并先建好名为 `zhiyuanwiki` 的
   Pages 项目。

## 重新生成数据

```sh
go run ./cmd/zhiyuan-data fenduan    # 一分一段 → JSON
go run ./cmd/zhiyuan-data yuanxiao   # 院校 / 院校×专业 / 2026 组视图
go run ./cmd/zhiyuan-data zhuanye    # 专业跨校聚合
go run ./cmd/zhiyuan-data dingwei    # 位次定位索引
```

## 数据来源与免责

数据源自黑龙江省招生考试信息港（lzk.hl.cn）官方公开数据，及万师兄·高考志愿填报大数据
（第三方整理）。本站数据**仅供参考**，一切以各省考试院及院校最新官方公布为准。
