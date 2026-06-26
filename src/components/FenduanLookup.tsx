import { useEffect, useState } from "preact/hooks";
import { scoreToRank, rankToScore, type YiFenYiDuan } from "../lib/fenduan";

type Mode = "s2r" | "r2s";
const LS_KEY = "fenduan.input"; // 记住上次的换算方向与输入值

export default function FenduanLookup({ table }: { table: YiFenYiDuan }) {
  const [mode, setMode] = useState<Mode>("s2r");
  const [val, setVal] = useState("");

  // 挂载后恢复上次输入；ready 之前不回写，避免用默认值覆盖已存值。
  const [ready, setReady] = useState(false);
  useEffect(() => {
    try {
      const raw = localStorage.getItem(LS_KEY);
      if (raw) {
        const s = JSON.parse(raw) as Partial<{ mode: Mode; val: string }>;
        if (s.mode === "s2r" || s.mode === "r2s") setMode(s.mode);
        if (typeof s.val === "string") setVal(s.val);
      }
    } catch {
      /* 隐私模式 / 损坏数据忽略 */
    }
    setReady(true);
  }, []);

  useEffect(() => {
    if (!ready) return;
    try {
      localStorage.setItem(LS_KEY, JSON.stringify({ mode, val }));
    } catch {
      /* 忽略 */
    }
  }, [ready, mode, val]);

  const n = parseInt(val, 10);
  const valid = !Number.isNaN(n) && n > 0;

  let result: string | null = null;
  if (valid) {
    if (mode === "s2r") {
      const r = scoreToRank(table, n);
      result = r == null ? null : `位次约 ${r.toLocaleString("zh-CN")}`;
    } else {
      const s = rankToScore(table, n);
      result = s == null ? null : `约 ${s} 分`;
    }
  }

  const tab = (m: Mode, label: string) => (
    <button
      type="button"
      onClick={() => setMode(m)}
      class={`rounded-md px-3 py-1.5 text-sm font-medium ${
        mode === m ? "bg-slate-900 text-white" : "bg-slate-100 text-slate-600 hover:bg-slate-200"
      }`}
    >
      {label}
    </button>
  );

  return (
    <div class="rounded-xl border border-slate-200 bg-white p-5">
      <div class="flex gap-2">
        {tab("s2r", "分数 → 位次")}
        {tab("r2s", "位次 → 分数")}
      </div>

      <div class="mt-4 flex items-center gap-3">
        <input
          type="number"
          inputMode="numeric"
          value={val}
          onInput={(e) => setVal((e.target as HTMLInputElement).value)}
          placeholder={mode === "s2r" ? "输入你的分数" : "输入你的位次"}
          class="w-44 rounded-md border border-slate-300 px-3 py-2 text-base focus:border-slate-500 focus:outline-none"
        />
        <span class="text-lg font-semibold text-slate-900">
          {valid ? (result ?? "—") : <span class="text-slate-400">↑ 待输入</span>}
        </span>
      </div>

      <p class="mt-3 text-xs text-slate-500">
        基于 {table.province} {table.year} {table.track}类官方一分一段表（{table.entries.length} 个分数段）。
        位次 = 全省该科类得分 ≥ 你的人数。
      </p>
    </div>
  );
}
