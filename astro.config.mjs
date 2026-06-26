// @ts-check
import { defineConfig } from "astro/config";
import preact from "@astrojs/preact";
import tailwindcss from "@tailwindcss/vite";

// 静态站，托管在 Cloudflare Pages（域名根路径）。见 docs/adr/0006。
// 对称 /[prov]/ 路由：根路径由 src/pages/index.astro 跳默认省；旧单省 URL 由
// public/_redirects 在 CF 边缘 301 到 /hlj/...（见 ADR-0009）。
export default defineConfig({
  site: "https://zhiyuanwiki.pages.dev",
  integrations: [preact()],
  vite: {
    plugins: [tailwindcss()],
  },
});
