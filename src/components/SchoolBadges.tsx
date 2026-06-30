import { useEffect } from "preact/hooks";
import { classify, type Bucket } from "../lib/dingwei";

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

    // 冲稳保角标
    let V = 0;
    let myTrack = "";
    try {
      V = parseInt(localStorage.getItem(`myRank.${prov}`) || "0", 10) || 0;
      myTrack = localStorage.getItem(`myTrack.${prov}`) || "";
    } catch {
      /* ignore */
    }
    if (V <= 0) return;
    document.querySelectorAll<HTMLElement>(".rank-badge").forEach((el) => {
      const r = parseInt(el.dataset.rank || "0", 10);
      const tr = el.dataset.track || "";
      if (r <= 0 || (myTrack && tr && myTrack !== tr)) return;
      const b = classify(V, r);
      if (!b) return; // 无效输入：不渲染角标
      el.textContent = `按你的位次 ${V.toLocaleString()}：${b}`;
      el.className = `rank-badge ml-2 rounded px-2 py-0.5 text-xs font-medium ${cls[b]}`;
    });

    // 招生专业排行：给每个 .reach-tag 填短标签 + 底色，并给该行左边框上色。
    document.querySelectorAll<HTMLElement>(".reach-tag").forEach((el) => {
      const r = parseInt(el.dataset.rank || "0", 10);
      const tr = el.dataset.track || "";
      if (r <= 0 || (myTrack && tr && myTrack !== tr)) return;
      const b = classify(V, r);
      if (!b) return;
      el.textContent = b;
      el.style.background = reachColor[b];
      el.closest<HTMLElement>(".yx-rank-row")?.style.setProperty("border-left-color", reachColor[b]);
    });

    // 「你的位次」分界线：插到排行里第一条「位次 ≥ 你」的专业之前（以上更难、以下托底）。
    const list = document.querySelector<HTMLElement>(".yx-ranklist");
    const you = list?.querySelector<HTMLElement>(".yx-youhere");
    const youV = you?.querySelector<HTMLElement>(".yx-youhere-v");
    if (list && you && youV && !(myTrack && list.dataset.track && myTrack !== list.dataset.track)) {
      const rows = Array.from(list.querySelectorAll<HTMLElement>(".yx-rank-row"));
      const target = rows.find((row) => parseInt(row.dataset.rank || "0", 10) >= V);
      if (target) list.insertBefore(you, target);
      else {
        const caps = list.querySelectorAll<HTMLElement>(".yx-rank-cap");
        list.insertBefore(you, caps[caps.length - 1]); // 比所有专业都易 → 落到「更易考 ↓」前
      }
      youV.textContent = `你的位次 ${V.toLocaleString()}`;
      you.hidden = false;
    }
  }, [prov]);
  return null;
}
