// 渲染浙江「位次定位」结果，验证三档=「按排位上下取个数」而非按百分比：
// 把过滤后的可填项按等效位次排好，稳=最贴你水平的 100 个（上下各半），
// 冲=再往上 100 个（更难），保=再往下 100 个（更易）。复现用户场景：位次 52,529。
// 关键：保不再是“位次比你差 10%+”才算（旧逻辑下保从低你 5,800 起、留下大空档），
// 而是紧贴你下方的 100 个，低你只有几百位。
import { afterAll, beforeAll, expect, test } from "vitest";
import { startPreview, renderToImage, type Preview } from "./render-glue";

let server: Preview;
beforeAll(async () => {
  server = await startPreview(4392);
}, 40_000);
afterAll(() => server?.stop());

// 三档列容器：同时带 overflow-hidden 与 rounded-xl 的只有这三列；按各自副标题定位。
const COL = "div.overflow-hidden.rounded-xl";
const nearestLowerBy = (text: string): number => {
  const m = text.match(/低你 ([\d,]+)/); // 升序排列，首个“低你”即最贴你的
  return m ? parseInt(m[1].replace(/,/g, ""), 10) : Infinity;
};

test(
  "浙江定位三档按排位上下各取 100（保紧贴你下方，无百分比空档）",
  async () => {
    const { page, browser, out } = await renderToImage({
      baseURL: server.baseURL,
      name: "locator-zj-window",
      path: "/zj/",
      viewport: { width: 1280, height: 1600 },
      fullPage: false,
      interact: async (p) => {
        await p.getByRole("button", { name: "位次", exact: true }).click();
        await p.getByPlaceholder("输入位次", { exact: true }).fill("52529");
        await p.getByText("你的全省位次").waitFor({ timeout: 8_000 });
        await p.waitForFunction(
          () => document.querySelectorAll('a[href^="/zj/yuanxiao/"]').length >= 300,
          { timeout: 8_000 },
        );
      },
    });

    const congCol = page.locator(COL).filter({ hasText: "够一够" });
    const wenCol = page.locator(COL).filter({ hasText: "较稳妥" });
    const baoCol = page.locator(COL).filter({ hasText: "兜得住" });
    const cardsIn = (col: typeof congCol) => col.locator('a[href^="/zj/yuanxiao/"]').count();

    // 该位次上下都 >150 个可填项 → 三档各满 100。
    expect(await cardsIn(congCol)).toBe(100);
    expect(await cardsIn(wenCol)).toBe(100);
    expect(await cardsIn(baoCol)).toBe(100);

    const congText = await congCol.innerText();
    const wenText = await wenCol.innerText();
    const baoText = await baoCol.innerText();

    // 冲全在你之上（更难）：只有“高你”，无“低你”。
    expect(congText).toContain("高你");
    expect(congText).not.toContain("低你");
    // 保全在你之下（更易）：只有“低你”，无“高你”。
    expect(baoText).toContain("低你");
    expect(baoText).not.toContain("高你");
    // 稳跨你水平：既有“高你”也有“低你”。
    expect(wenText).toContain("高你");
    expect(wenText).toContain("低你");

    // 核心修复：保紧贴你下方——最近的保底学校低你只有几百位，而非旧逻辑的 5,800+。
    const nearestBao = nearestLowerBy(baoText);
    expect(nearestBao).toBeLessThan(3000);

    // 列头是干净的计数“100 个”，不再有按百分比截断的“前 N”标记。
    for (const t of [congText, wenText, baoText]) {
      expect(t).toContain("100 个");
      expect(t).not.toContain("前 ");
    }

    console.log(`冲/稳/保 各 100 · 最近保底 低你 ${nearestBao} → ${out}`);

    await browser.close();
  },
  60_000,
);
