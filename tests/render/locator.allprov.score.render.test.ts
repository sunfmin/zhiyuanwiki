// 渲染冒烟（全省）：每个省都用「分数」输入，确认下方真的渲染出可填报专业。
// 防回归——河南曾报「输入分数后下方空白」。用真实数据：取该省「定位科类」一分一段的中位分
//（rank 省按分数→位次换算），西藏无一分一段→取其 locator 录取最低分中位；喂进真实页面后，
// 断言冲/稳/保「主档」（排除「仅供参考」远档折叠区）里有至少一个专业链接。
//
// 前置：需先 `npm run build`（preview 服务 dist/）。
import { afterAll, beforeAll, describe, expect, test } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { chromium, type Browser } from "playwright";
import { startPreview, type Preview } from "./render-glue";
import { PROVINCE_SLUGS, provinceConfig, trackSlugOf } from "../../src/lib/provinces";
import { rankToScore, type YiFenYiDuan } from "../../src/lib/fenduan";

const median = (xs: number[]): number => (xs.length ? [...xs].sort((a, b) => a - b)[Math.floor(xs.length / 2)] : 0);

// 取一个「站在可填报专业正中间」的代表分——比按全省人数取中位更稳：含专科续段的省（山东/河南）人数
// 中位会落到本科线附近，保档塌空。故用 locator 里真实可填条目的等效位次（rank 省）/录取最低分（西藏）
// 的中位，再换回分数。这样模拟的考生必然处在可填档的正中，冲/稳/保三列都应有货。
function representativeScore(prov: string): number {
  const cfg = provinceConfig(prov);
  const slug = trackSlugOf(cfg, cfg.tracks[0].name);
  const loc = JSON.parse(
    readFileSync(resolve(`public/data/${prov}/locator-${slug}.json`), "utf8"),
  ) as { r?: number; s?: number }[];
  if (cfg.locatorBasis === "score") {
    // 西藏：无一分一段，直接用录取最低分(s)中位作为输入分。
    return median(loc.map((e) => e.s).filter((x): x is number => typeof x === "number" && x > 0));
  }
  // 位次省：取可填条目等效位次(r)的中位，经一分一段换回分数（fenduanTrack 表，与默认科类同轨）。
  const medR = median(loc.map((e) => e.r).filter((x): x is number => typeof x === "number" && x > 0));
  const fdSlug = trackSlugOf(cfg, cfg.fenduanTrack);
  const tbl = JSON.parse(
    readFileSync(resolve(`src/data/${prov}/yifenyiduan/${fdSlug}-${cfg.fenduanYear}.json`), "utf8"),
  ) as YiFenYiDuan;
  return rankToScore(tbl, medR) ?? 0;
}

let server: Preview;
let browser: Browser;
beforeAll(async () => {
  server = await startPreview();
  browser = await chromium.launch();
}, 60_000);
afterAll(async () => {
  await browser?.close();
  server?.stop();
});

describe("每省：分数输入 → 冲/稳/保三档都渲染出可填报专业", () => {
  test.each(PROVINCE_SLUGS)(
    "%s",
    async (prov) => {
      const cfg = provinceConfig(prov);
      const score = representativeScore(prov);
      expect(score, `${cfg.name}(${prov}) 取不到代表分——数据文件缺失？`).toBeGreaterThan(0);

      const page = await browser.newPage({ viewport: { width: 1180, height: 1200 } });
      try {
        await page.goto(server.baseURL + `/${prov}/`, { waitUntil: "networkidle" });
        await page.getByPlaceholder("输入分数", { exact: true }).fill(String(score));

        // 主档（非 details 远档预览）里出现专业链接 = 下方有数据。
        const mainCount = () =>
          page.evaluate(
            (p) =>
              [...document.querySelectorAll(`a[href^="/${p}/yuanxiao/"]`)].filter((a) => !a.closest("details")).length,
            prov,
          );
        // 三列里若某列空，会渲染「这一档暂无可填」<li>。代表分下三列都应有货 → 计数应为 0。
        const emptyTiers = () =>
          page.evaluate(() => [...document.querySelectorAll("li")].filter((li) => /这一档暂无可填/.test(li.textContent ?? "")).length);

        await page
          .waitForFunction(
            (p) =>
              [...document.querySelectorAll(`a[href^="/${p}/yuanxiao/"]`)].filter((a) => !a.closest("details")).length > 0,
            prov,
            { timeout: 12_000 },
          )
          .catch(() => {
            /* 超时则下方断言给出可读的失败信息 */
          });

        const [cards, empties] = [await mainCount(), await emptyTiers()];
        await page.screenshot({ path: resolve("out", `loc-allprov-${prov}.png`), fullPage: false });
        expect(cards, `${cfg.name}(${prov}) 输入分数 ${score} 后，主档下方无任何可填报专业`).toBeGreaterThan(0);
        expect(
          empties,
          `${cfg.name}(${prov}) 输入分数 ${score}：有 ${empties} 个档位显示「这一档暂无可填」——代表分下冲/稳/保应都有货`,
        ).toBe(0);
      } finally {
        await page.close();
      }
    },
    30_000,
  );
});
