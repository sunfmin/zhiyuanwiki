import { useEffect } from "preact/hooks";
import { classify, classifyScore, type Bucket } from "../lib/dingwei";
import { provinceConfig } from "../lib/provinces";

const cls: Record<Bucket, string> = {
  够不着: "bg-slate-100 text-slate-500",
  冲: "bg-rose-100 text-rose-700",
  稳: "bg-amber-100 text-amber-700",
  保: "bg-emerald-100 text-emerald-700",
  过保: "bg-slate-100 text-slate-500",
};

// 冲/稳/保 短标签底色（与「位次定位」配色基调一致；够不着/过保走灰）
const reachColor: Record<Bucket, string> = {
  够不着: "#9a948a",
  冲: "#c0202e",
  稳: "#b9892b",
  保: "#2f7d5b",
  过保: "#9a948a",
};

/**
 * 单个 island：读 localStorage 里的访客位次，给本页所有 .rank-badge（带 data-rank/data-track）
 * 就地填上冲/稳/保；给招生专业排行的 .reach-tag 填上短标签 + 底色 + 给该行左边框上色；
 * 并把「你的位次」分界线 (.yx-youhere) 插进排行里对应档位（以上更难、以下托底）。
 * 还在 URL 带 #z-{专业键} 时自动展开对应专业区块并滚动到位。
 * 用一个 island 注解整页，避免每个专业各挂一个 island。
 */
export default function SchoolBadges({ prov }: { prov: string }) {
  useEffect(() => {
    // 锚点：展开并滚动到目标 <details>
    const hash = decodeURIComponent(location.hash.replace(/^#/, ""));
    if (hash) {
      const el = document.getElementById(hash);
      if (el instanceof HTMLDetailsElement) {
        el.open = true;
        el.scrollIntoView({ block: "start" });
      }
    }

    // 只有分数省（西藏）：访客存的是分数（myScore），按数据元素的 data-score 用绝对分差判档；
    // 其余省按位次（myRank）+ data-rank 用比值判档。两条路径只是「读哪个键 + 用哪个 classify」之别。
    const scoreMode = provinceConfig(prov).locatorBasis === "score";
    let V = 0;
    let myTrack = "";
    try {
      V = parseInt(localStorage.getItem(`${scoreMode ? "myScore" : "myRank"}.${prov}`) || "0", 10) || 0;
      myTrack = localStorage.getItem(`myTrack.${prov}`) || "";
    } catch {
      /* ignore */
    }
    if (V <= 0) return;
    // bucketOf：从元素的 data-rank/data-score 取该专业的「值」，判出冲稳保（科类不符或无值→null）。
    const bucketOf = (el: HTMLElement): Bucket | null => {
      const tr = el.dataset.track || "";
      if (myTrack && tr && myTrack !== tr) return null;
      if (scoreMode) {
        const s = parseInt(el.dataset.score || "0", 10);
        return s > 0 ? classifyScore(V, s) : null;
      }
      const r = parseInt(el.dataset.rank || "0", 10);
      return r > 0 ? classify(V, r) : null;
    };
    const vLabel = scoreMode ? `${V} 分` : V.toLocaleString();

    document.querySelectorAll<HTMLElement>(".rank-badge").forEach((el) => {
      const b = bucketOf(el);
      if (!b) return; // 无效输入 / 科类不符：不渲染角标
      el.textContent = `按你的${scoreMode ? "分数" : "位次"} ${vLabel}：${b}`;
      el.className = `rank-badge ml-2 rounded px-2 py-0.5 text-xs font-medium ${cls[b]}`;
    });

    // 招生专业排行：给每个 .reach-tag 填短标签 + 底色，并给该行左边框上色。
    document.querySelectorAll<HTMLElement>(".reach-tag").forEach((el) => {
      const b = bucketOf(el);
      if (!b) return;
      el.textContent = b;
      el.style.background = reachColor[b];
      el.closest<HTMLElement>(".yx-rank-row")?.style.setProperty("border-left-color", reachColor[b]);
    });

    // 「你在这里」分界线：插到排行里第一条「不比你难」的专业之前（以上更难、以下托底）。
    // 位次省：第一条 data-rank ≥ V（位次更大=更易）；只有分数省：第一条 data-score ≤ V（分更低=更易）。
    const list = document.querySelector<HTMLElement>(".yx-ranklist");
    const you = list?.querySelector<HTMLElement>(".yx-youhere");
    const youV = you?.querySelector<HTMLElement>(".yx-youhere-v");
    if (list && you && youV && !(myTrack && list.dataset.track && myTrack !== list.dataset.track)) {
      const rows = Array.from(list.querySelectorAll<HTMLElement>(".yx-rank-row"));
      const target = rows.find((row) =>
        scoreMode
          ? parseInt(row.dataset.score || "0", 10) <= V
          : parseInt(row.dataset.rank || "0", 10) >= V,
      );
      if (target) list.insertBefore(you, target);
      else {
        const caps = list.querySelectorAll<HTMLElement>(".yx-rank-cap");
        list.insertBefore(you, caps[caps.length - 1]); // 比所有专业都易 → 落到末尾「更易」前
      }
      youV.textContent = `你的${scoreMode ? "分数" : "位次"} ${vLabel}`;
      you.hidden = false;
    }
  }, [prov]);
  return null;
}
