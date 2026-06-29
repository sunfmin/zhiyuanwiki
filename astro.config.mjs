// @ts-check
import { defineConfig } from "astro/config";
import preact from "@astrojs/preact";
import tailwindcss from "@tailwindcss/vite";

// 静态站，托管在 Cloudflare R2 + Rules（自定义域 zhiyuanwiki.com 根路径）。见 docs/adr/0018（推翻 0006 的 Pages）。
// trailingSlash:'always'：Rules-only 下目录式 URL（以 / 结尾）由一条 URL Rewrite 补 index.html，
//   故所有站内链接必须带尾斜杠，否则命中 R2 404（无 Worker 兜底）。由 tests/render 的尾斜杠不变量测试守护。
// 对称 /[prov]/ 路由：根路径 src/pages/index.astro 是省份列表落地页（全 31 省，见 ADR-0016）；
// 旧单省 URL 由 Cloudflare Bulk Redirects 在边缘 301 到 /hlj/...（见 ADR-0009；R2 不读 public/_redirects）。
export default defineConfig({
  site: "https://zhiyuanwiki.com",
  trailingSlash: "always",
  integrations: [preact()],
  vite: {
    plugins: [tailwindcss()],
  },
});
