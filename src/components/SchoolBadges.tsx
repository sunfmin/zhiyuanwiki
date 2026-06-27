import { useEffect } from "preact/hooks";
import { classify, type Bucket } from "../lib/dingwei";

const cls: Record<Bucket, string> = {
  够不着: "bg-slate-100 text-slate-500",
  冲: "bg-rose-100 text-rose-700",
  稳: "bg-amber-100 text-amber-700",
  保: "bg-emerald-100 text-emerald-700",
  过保: "bg-slate-100 text-slate-500",
};

/**
 * 单个 island：读 localStorage 里的访客位次，给本页所有 .rank-badge（带 data-rank/data-track）
 * 就地填上冲/稳/保。并在 URL 带 #z-{专业键} 时自动展开对应专业区块并滚动到位。
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
  }, [prov]);
  return null;
}
