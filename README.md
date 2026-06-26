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

## 数据来源与免责

数据源自黑龙江省招生考试信息港（lzk.hl.cn）官方公开数据，及万师兄·高考志愿填报大数据
（第三方整理）。本站数据**仅供参考**，一切以各省考试院及院校最新官方公布为准。
