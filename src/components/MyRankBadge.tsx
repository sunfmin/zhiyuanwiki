import { useEffect, useState } from "preact/hooks";
import { classify, type Bucket } from "../lib/dingwei";

const cls: Record<Bucket, string> = {
  冲: "bg-rose-100 text-rose-700",
  稳: "bg-amber-100 text-amber-700",
  保: "bg-emerald-100 text-emerald-700",
  out: "bg-slate-100 text-slate-500",
};

/**
 * 读取 localStorage 里的访客位次（由位次定位工具写入），与本专业某科类的录取位次比较，
 * 在院校/专业页就地显示冲/稳/保角标。科类不一致则不显示。
 */
export default function MyRankBadge({ rank, track }: { rank: number; track: string }) {
  const [V, setV] = useState(0);
  const [myTrack, setMyTrack] = useState("");
  useEffect(() => {
    try {
      setV(parseInt(localStorage.getItem("myRank") || "0", 10) || 0);
      setMyTrack(localStorage.getItem("myTrack") || "");
    } catch {
      /* ignore */
    }
  }, []);

  if (V <= 0 || rank <= 0) return null;
  if (myTrack && track && myTrack !== track) return null;

  const b = classify(V, rank);
  const label = b === "out" ? "够不着" : b;
  return (
    <span class={`ml-2 rounded px-2 py-0.5 text-xs font-medium ${cls[b]}`}>
      按你的位次 {V.toLocaleString()}：{label}
    </span>
  );
}
