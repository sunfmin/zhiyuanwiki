// @ts-check
import { defineConfig } from "astro/config";
import preact from "@astrojs/preact";
import tailwindcss from "@tailwindcss/vite";

// 静态站。生产由 Cloudflare R2 托管：apex zhiyuanwiki.com 与 r2.zhiyuanwiki.com 皆挂 R2 自定义域，
//   前置一条 host-scoped URL Rewrite 补目录 index.html——R2 自身不解析目录 index.html（r2.dev 与自定义域
//   皆 404 于目录，Cloudflare 官方限制）。已退役 Pages（2026-06-29 切 apex 到 R2）。见 docs/adr/0018、0019、0020。
// trailingSlash:'always' 两个理由（仍成立）：(a) 构建必须目录式（dist/zj/index.html），R2 才有 zj/index.html
//   这个 key；(b) 站内链接带尾斜杠，才命中「path 以 / 结尾 → 补 index.html」的 Rewrite（无斜杠 /zj 命中 R2 404）。
//   由 tests/render 尾斜杠不变量测试守护。
//   ⚠ 这条补 index.html 的 Rewrite 必须 host-scoped 到「纯 R2」域（现为 zhiyuanwiki.com + r2.zhiyuanwiki.com）。
//      切勿施于任何 Pages 服务的 host：Pages 会把 …/index.html 308 回 …/，与补 index.html 打架成无限重定向
//      （ERR_TOO_MANY_REDIRECTS，2026-06-29 曾致宕机——当时 ADR-0019 误判为 R2 行为，实为 Pages）。见 ADR-0020。
// 对称 /[prov]/ 路由：根路径 src/pages/index.astro 是省份列表落地页（全 31 省，见 ADR-0016）。
export default defineConfig({
  site: "https://zhiyuanwiki.com",
  trailingSlash: "always",
  integrations: [preact()],
  vite: {
    plugins: [tailwindcss()],
  },
});
