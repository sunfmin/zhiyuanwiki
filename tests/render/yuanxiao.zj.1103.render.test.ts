// 渲染浙江「院校页面」北京大学（/zj/yuanxiao/1103/）→ out/yuanxiao-1103.png。
// 北大在浙江是全省最难考的一档：最好的专业要全省第 5 名（元培 2025），工商管理类 2023 曾录取全省第 1。
// 这个页面是「位次标尺」签名件的最佳例子：把 2026 招生专业按往年最低位次摆上一条共享对数轴，
// 再用访客存入的位次落一条「你的位次」朱线，冲/稳/保就此变成空间关系。
// 端到端验证：真实 src/data/zj/schools/1103.json → 真实页面逻辑（顶点读数 + 位次标尺 + 冲稳保角标 + 走势）。
// 见 ADR-0009（多省份泛化）与 ADR-0003（院校为稳定主线）。
import { afterAll, beforeAll, expect, test } from "vitest";
import { startPreview, renderToImage, type Preview } from "./render-glue";

let server: Preview;
beforeAll(async () => {
  server = await startPreview(4392);
}, 40_000);
afterAll(() => server?.stop());

test(
  "浙江院校页面（北京大学 · 位次标尺 + 你的位次线 + 历年录取位次）",
  async () => {
    const { page, browser, out } = await renderToImage({
      baseURL: server.baseURL,
      name: "yuanxiao-1103",
      path: "/zj/yuanxiao/北京大学/",
      viewport: { width: 1120, height: 1600 },
      fullPage: true,
      interact: async (p) => {
        await p.getByRole("heading", { level: 1, name: "北京大学" }).waitFor({ timeout: 8_000 });
        // 模拟访客在「位次定位」存过位次 95（落在 5–100 标尺区间内）：
        // 重载后 island 给每个专业 summary 填冲/稳/保，并在标尺上落「你的位次」朱线。
        await p.evaluate(() => {
          localStorage.setItem("myRank.zj", "95");
          localStorage.setItem("myTrack.zj", "综合");
        });
        await p.reload({ waitUntil: "networkidle" });
        await p.getByRole("heading", { level: 1, name: "北京大学" }).waitFor({ timeout: 8_000 });
        // 等「你的位次」分界线被 island 插进排行并显形（去掉 hidden 即出现）。
        await p.locator(".yx-youhere:not([hidden])").first().waitFor({ timeout: 8_000 });
        // 展开前 3 个专业，让历年位次表 + 走势 sparkline 进截图（默认折叠）。
        const details = p.locator("main details");
        const n = Math.min(3, await details.count());
        for (let i = 0; i < n; i++) {
          await details.nth(i).locator("summary").click();
        }
        await p
          .locator("main details[open] svg polyline")
          .first()
          .waitFor({ timeout: 8_000 })
          .catch(() => {
            /* 头 3 个专业可能只有 1 年数据，没有折线也可接受 */
          });
      },
    });

    const main = page.locator("main");
    const mainText = await main.innerText();

    // 1) 院校身份头：真实数据流到 UI（院校名 + 院校代码 + 由真实逻辑算出的两个计数）。
    //    换用一致源（ADR-0022）后北大在浙江收录 3 个招生专业、3 个专业有往年录取。
    expect(mainText).toContain("北京大学");
    expect(mainText).toContain("院校代码 1103");
    expect(mainText).toMatch(/3\s*个 2026 招生专业/);
    expect(mainText).toMatch(/3\s*个专业有 2022–2025 录取记录/);

    // 2) 顶点读数（首屏主角）：最难考的专业 = 全省第 5 名（理科试验班类 2025），并给出录取位次区间 + 分数。
    expect(mainText).toContain("最难考的专业");
    expect((await main.locator(".yx-apex-val").innerText()).trim()).toBe("5");
    expect(await main.locator(".yx-apex-major").innerText()).toContain("理科试验班类");
    expect(await main.locator(".yx-apex-range").innerText()).toContain("录取位次 5–92");

    // 3) 浙江走「招生专业」视图（major 模型），而非黑龙江的「院校专业组」。
    expect(mainText).toContain("2026 报考视图（招生专业）");
    expect(mainText).not.toContain("院校专业组）");

    // 4) 签名件「你在这里」排行：3 个专业全有往年位次，按从难到易排成 3 行，各按 #z-<majorKey> 锚到历年区。
    expect(await main.locator(".yx-ranklist").count()).toBe(1);
    expect(await main.locator(".yx-ranklist .yx-rank-row").count()).toBe(3);
    expect(await main.locator('a[href^="#z-"]').count()).toBe(3);
    const listText = await main.locator(".yx-ranklist").innerText();
    expect(listText).toContain("位次 5");
    expect(listText).toContain("位次 92");

    // 4b) 「你的位次」分界线插进排行并显形，标签带存入的位次 95；每行冲/够不着 短标签也被 island 填上
    //     （V=95 下：文科试验班类 92=冲；理科试验班类 5 / 工商管理类 24 = 够不着）。
    expect(await main.locator(".yx-youhere:not([hidden])").count()).toBe(1);
    expect(await main.locator(".yx-youhere-v").innerText()).toContain("你的位次 95");
    const tags = await main.locator(".yx-ranklist .reach-tag").allInnerTexts();
    expect(tags).toContain("冲");
    expect(tags).toContain("够不着");

    // 5) 历年录取位次：leaves 渲染成 3 个可展开 <details>；浙江单科类「综合」不出现冗余的「综合类」分头。
    expect(mainText).toContain("全部专业 · 历年录取位次");
    expect(await main.locator("details").count()).toBe(3);
    expect(await main.locator("details h3").count()).toBe(0);
    // 冲/够不着角标改挂到每个专业的 summary 行（无需展开即可见），恰好 3 条。
    expect(await main.locator("summary .rank-badge").count()).toBe(3);
    // island 按存入的位次 95 给 summary 角标就地填值：文科试验班类（92）=冲、顶尖两专业（5/24）=够不着。
    expect(mainText).toContain("按你的位次 95：冲");
    expect(mainText).toContain("按你的位次 95：够不着");

    // 6) 抽样真实专业名确实出现在页面上。
    expect(mainText).toContain("工商管理类");
    expect(mainText).toContain("文科试验班类");

    console.log(`北京大学院校页：标尺 3 rung · 历年 3 专业 · 你的位次线就位 → ${out}`);

    await browser.close();
  },
  60_000,
);
