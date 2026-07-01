// /sitemap.xml 别名不变量：@astrojs/sitemap 只产 sitemap-index.xml，构建末由 astro.config 的
// sitemap-xml-alias 整合复制出一份 sitemap.xml（多数第三方工具/爬虫直接探 /sitemap.xml）。
// 断言 dist/sitemap.xml 存在、非空、且内容与 sitemap-index.xml 逐字节一致——守护这条可发现性，
// 并挡住别名整合悄悄失效或与真索引漂移（如日后 URL 超 45000 分片，副本须一并继承）。
// 纯文件比对，不起浏览器；放 tests/render/ 仅因它依赖 `npm run build` 的产物。
import { expect, test } from "vitest";
import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";

test("dist/sitemap.xml 存在且 == sitemap-index.xml（/sitemap.xml 别名不变量）", () => {
  const index = resolve("dist/sitemap-index.xml");
  const alias = resolve("dist/sitemap.xml");
  expect(existsSync(index), "缺 dist/sitemap-index.xml——是否漏跑 npm run build？").toBe(true);
  expect(existsSync(alias), "缺 dist/sitemap.xml——sitemap-xml-alias 整合未生效？").toBe(true);

  const src = readFileSync(index, "utf8");
  expect(src.length, "sitemap-index.xml 为空").toBeGreaterThan(0);
  expect(readFileSync(alias, "utf8")).toBe(src);
});
