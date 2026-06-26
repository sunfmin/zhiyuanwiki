// 渲染浙江「院校页面」→ out/yuanxiao-zj.png。
// 用浙江大学（/zj/yuanxiao/0001/）做例子：浙江=专业平行志愿（fillModel=major），
// 故院校页走「招生专业」视图（每个院校×专业是独立投档单位，无组内调剂）。
// 端到端验证：真实 src/data/zj/schools/0001.json → 真实页面逻辑（计划表 + 历年位次 + 走势 sparkline）。
// 见 ADR-0009（多省份泛化）与 ADR-0003（院校为稳定主线）。
import { afterAll, beforeAll, expect, test } from "vitest";
import { startPreview, renderToImage, type Preview } from "./render-glue";

let server: Preview;
beforeAll(async () => {
  server = await startPreview(4391);
}, 40_000);
afterAll(() => server?.stop());

test(
  "浙江院校页面（浙江大学 · 招生专业视图 · 历年录取位次）",
  async () => {
    const { page, browser, out } = await renderToImage({
      baseURL: server.baseURL,
      name: "yuanxiao-zj",
      path: "/zj/yuanxiao/0001/",
      viewport: { width: 1120, height: 1600 },
      fullPage: true,
      interact: async (p) => {
        // 院校名先到，确认真实数据已就绪。
        await p.getByRole("heading", { level: 1, name: "浙江大学" }).waitFor({ timeout: 8_000 });
        // 展开前 3 个专业，让历年位次表 + 走势 sparkline 在截图里可见（默认折叠）。
        const details = p.locator("main details");
        const n = Math.min(3, await details.count());
        for (let i = 0; i < n; i++) {
          await details.nth(i).locator("summary").click();
        }
        // 等到至少一条走势折线（>=2 年的专业才画 svg polyline）渲染出来。
        await p
          .locator("main details[open] svg polyline")
          .first()
          .waitFor({ timeout: 8_000 })
          .catch(() => {
            /* 头 3 个专业可能都只有 1 年数据，没有折线也可接受 */
          });
      },
    });

    const main = page.locator("main");
    const mainText = await main.innerText();

    // 1) 真实数据流到了 UI：院校名 + 院校代码摘要行（unitCount/leaves 由真实逻辑算出）。
    expect(mainText).toContain("浙江大学");
    expect(mainText).toContain("院校代码 0001");
    expect(mainText).toMatch(/26\s*个 2026 招生专业/);
    expect(mainText).toMatch(/32\s*个专业有 2022–2025 录取记录/);

    // 2) 浙江走「招生专业」视图（major 模型），而非黑龙江的「院校专业组」。
    expect(mainText).toContain("2026 报考视图（招生专业）");
    expect(mainText).not.toContain("院校专业组）");
    // 计划表里每个专业名锚到 #z-<majorKey>：恰好 plan2026 的 26 条。
    const planRows = await main.locator('table a[href^="#z-"]').count();
    expect(planRows).toBe(26);

    // 3) 历年录取位次区块：leaves 渲染成 32 个可展开 <details>。
    expect(mainText).toContain("全部专业 · 历年录取位次");
    const leafCount = await main.locator("details").count();
    expect(leafCount).toBe(32);

    // 4) 抽样真实专业名（分别来自 plan2026 与 leaves）确实出现在页面上。
    expect(mainText).toContain("工科试验班（竺可桢学院图灵班）");
    expect(mainText).toContain("人工智能");

    console.log(`浙江大学院校页：${planRows} 个招生专业 · ${leafCount} 个历年专业 → ${out}`);

    await browser.close();
  },
  60_000,
);
