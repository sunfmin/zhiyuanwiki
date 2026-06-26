// @ts-check
import { defineConfig } from "astro/config";
import preact from "@astrojs/preact";
import tailwindcss from "@tailwindcss/vite";

// 静态站，托管在 Cloudflare Pages（域名根路径）。见 docs/adr/0006。
export default defineConfig({
  site: "https://zhiyuanwiki.pages.dev",
  // 位次定位已成为主页（/）；旧的 /dingwei 链接重定向到根，保留外部链接可用。
  redirects: {
    "/dingwei": "/",
    "/dingwei/": "/",
  },
  integrations: [preact()],
  vite: {
    plugins: [tailwindcss()],
  },
});
