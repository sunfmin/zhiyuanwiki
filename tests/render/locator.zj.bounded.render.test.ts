// 渲染浙江「位次定位」结果，验证冲稳保=「按把握比值分档」而非「按位次邻域凑数」(ADR-0010)。
// 复现用户场景：位次 52,529，并叠加「院校层次=211」让冲档变稀疏，从而能观察远档预览。关键回归：
//  - 冲只含真正够得着的（V/R≤1.15 → 最远高你约 13%），远超的进「够不着」仅供参考，不再混入冲；
//  - 每主档显示有上限（不再硬凑 100）；冲全在你之上、保全在你之下、稳跨你水平；
//  - 冲档稀疏（<100）时末尾补「够不着」预览，把远档降级而非误标成冲（密集时按 ADR 不补，是另一条正常路径）；
//  - 浙江有一分一段 → 给出等效分（约 X 分）。
import { afterAll, beforeAll, expect, test } from "vitest";
import { startPreview, renderToImage, type Preview } from "./render-glue";

let server: Preview;
beforeAll(async () => {
  server = await startPreview(4392);
}, 40_000);
afterAll(() => server?.stop());

// 三档列容器：同时带 overflow-hidden 与 rounded-xl 的只有这三列；按各自副标题定位。
const COL = "div.overflow-hidden.rounded-xl";
const maxHigherBy = (text: string): number => {
  // 所有「高你 N 位」取最大（=最远的冲）。
  let max = 0;
  for (const m of text.matchAll(/高你 ([\d,]+) 位/g)) {
    max = Math.max(max, parseInt(m[1].replace(/,/g, ""), 10));
  }
  return max;
};

test(
  "浙江定位按把握比值分档：冲有界 · 远档进够不着 · 给出等效分",
  async () => {
    const V = 52529;
    const { page, browser, out } = await renderToImage({
      baseURL: server.baseURL,
      name: "locator-zj-bounded",
      path: "/zj/",
      viewport: { width: 1280, height: 1600 },
      fullPage: false,
      interact: async (p) => {
        await p.getByRole("button", { name: "位次", exact: true }).click();
        await p.getByPlaceholder("输入位次", { exact: true }).fill(String(V));
        await p.getByText("你的全省位次").waitFor({ timeout: 8_000 });
        // 不加筛选时冲档本就 >100（够得着的足够多）→ 按 ADR-0010「真实档已 ≥100 则不补」末尾不挂远档。
        // 限定 211 让冲档变稀疏（<100），触发末尾「够不着」补齐预览，据此验证远档被降级而非误标成冲。
        await p.getByRole("button", { name: "211", exact: true }).first().click();
        // 等到「够不着」预览渲染出来（=筛选已生效且远档已挂），再做断言。
        await p.getByText("够不着").first().waitFor({ timeout: 8_000 });
      },
    });

    const congCol = page.locator(COL).filter({ hasText: "够一够" });
    const wenCol = page.locator(COL).filter({ hasText: "较稳妥" });
    const baoCol = page.locator(COL).filter({ hasText: "兜得住" });

    // 主列卡片（:scope > ul，不含 details 远档预览）有显示上限，不再硬凑 100。
    const mainCards = (col: typeof congCol) =>
      col.locator(":scope > ul > li a[href^='/zj/yuanxiao/']").count();
    expect(await mainCards(congCol)).toBeGreaterThan(0);
    expect(await mainCards(congCol)).toBeLessThanOrEqual(30);
    expect(await mainCards(wenCol)).toBeLessThanOrEqual(30);
    expect(await mainCards(baoCol)).toBeLessThanOrEqual(30);

    const congText = await congCol.innerText();
    const wenText = await wenCol.innerText();
    const baoText = await baoCol.innerText();

    // 方向：冲全在你之上、保全在你之下、稳跨你水平。
    expect(congText).toContain("高你");
    expect(baoText).toContain("低你");
    expect(wenText).toContain("高你");
    expect(wenText).toContain("低你");

    // 核心修复：冲列主区最远项仍在比值带内（高你 ≤ ~13%），不含远超的幻想院校。
    // V/R≤1.15 → 高你 ≤ V·(1−1/1.15) ≈ 6,852；留余量取 7,200。
    const congMainText = await congCol.locator(":scope > ul").innerText();
    const farthestCong = maxHigherBy(congMainText);
    expect(farthestCong).toBeGreaterThan(0);
    expect(farthestCong).toBeLessThanOrEqual(7200);

    // 够不着作为「仅供参考」预览挂在冲列末尾（远档被降级，而非误标成冲）。
    const farSummary = congCol.locator("details summary");
    expect(await farSummary.count()).toBe(1);
    const farText = await farSummary.innerText();
    expect(farText).toContain("够不着");
    expect(farText).toContain("仅供参考");

    // 浙江有一分一段 → 等效分（约 X 分）。
    expect(congText).toContain("等效分");
    expect(congText).toMatch(/约 [\d,]+ 分/);

    console.log(`冲列主区最远 高你 ${farthestCong} 位（≤6852 带内）→ ${out}`);

    await browser.close();
  },
  60_000,
);
