import { useEffect, useMemo, useRef, useState } from "preact/hooks";
import { scoreToRank, type YiFenYiDuan } from "../lib/fenduan";
import { classify, selKeAllows, type Bucket, type LocEntry } from "../lib/dingwei";

type Track = "物理" | "历史";
const RESELECT = ["化学", "生物", "政治", "地理"] as const;
const CAP = 60;
const LS_KEY = "dingwei.input"; // 记住上次输入：科类 / 分或位次 / 再选科目

const trackFile: Record<Track, string> = {
  物理: "/data/locator-wuli.json",
  历史: "/data/locator-lishi.json",
};

export default function Locator({ wuliTable }: { wuliTable: YiFenYiDuan }) {
  const [track, setTrack] = useState<Track>("物理");
  const [mode, setMode] = useState<"score" | "rank">("score");
  const [val, setVal] = useState("");
  const [sel, setSel] = useState<Record<string, boolean>>({ 化学: true, 生物: true, 政治: false, 地理: false });

  const [entries, setEntries] = useState<LocEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const cache = useRef<Record<string, LocEntry[]>>({});

  // 历史类暂无 2026 一分一段，强制按位次输入。
  const effectiveMode: "score" | "rank" = track === "历史" ? "rank" : mode;

  // 挂载后恢复上次输入（localStorage）；ready 之前不回写，避免用默认值覆盖已存值。
  const [ready, setReady] = useState(false);
  useEffect(() => {
    try {
      const raw = localStorage.getItem(LS_KEY);
      if (raw) {
        const s = JSON.parse(raw) as Partial<{ track: Track; mode: "score" | "rank"; val: string; sel: Record<string, boolean> }>;
        if (s.track === "物理" || s.track === "历史") setTrack(s.track);
        if (s.mode === "score" || s.mode === "rank") setMode(s.mode);
        if (typeof s.val === "string") setVal(s.val);
        if (s.sel && typeof s.sel === "object") setSel((p) => ({ ...p, ...s.sel }));
      }
    } catch {
      /* 隐私模式 / 损坏数据忽略 */
    }
    setReady(true);
  }, []);

  useEffect(() => {
    if (!ready) return;
    try {
      localStorage.setItem(LS_KEY, JSON.stringify({ track, mode, val, sel }));
    } catch {
      /* 忽略 */
    }
  }, [ready, track, mode, val, sel]);

  useEffect(() => {
    const file = trackFile[track];
    if (cache.current[track]) {
      setEntries(cache.current[track]);
      return;
    }
    setLoading(true);
    fetch(file)
      .then((r) => r.json())
      .then((d: LocEntry[]) => {
        cache.current[track] = d;
        setEntries(d);
      })
      .finally(() => setLoading(false));
  }, [track]);

  const chosen = useMemo(() => {
    const s = new Set<string>([track]);
    for (const k of RESELECT) if (sel[k]) s.add(k);
    return s;
  }, [track, sel]);

  // 访客位次
  const V = useMemo(() => {
    const n = parseInt(val, 10);
    if (Number.isNaN(n) || n <= 0) return 0;
    if (effectiveMode === "rank") return n;
    return scoreToRank(wuliTable, n) ?? 0;
  }, [val, effectiveMode, wuliTable]);

  useEffect(() => {
    if (V > 0) {
      try {
        localStorage.setItem("myRank", String(V));
        localStorage.setItem("myTrack", track);
      } catch {
        /* 隐私模式等忽略 */
      }
    }
  }, [V, track]);

  const buckets = useMemo(() => {
    const out: Record<Bucket, LocEntry[]> = { 冲: [], 稳: [], 保: [], out: [] };
    if (V <= 0) return out;
    for (const e of entries) {
      if (!selKeAllows(e.sk, chosen)) continue;
      const b = classify(V, e.r);
      if (b !== "out") out[b].push(e);
    }
    for (const k of ["冲", "稳", "保"] as Bucket[]) {
      out[k].sort((a, b) => a.r - b.r);
    }
    return out;
  }, [entries, V, chosen]);

  const cfg: { key: Bucket; meaning: string; bar: string; label: string; delta: string }[] = [
    { key: "冲", meaning: "够一够 · 偏难", bar: "bg-rose-500", label: "text-rose-700", delta: "text-rose-600" },
    { key: "稳", meaning: "较稳妥 · 匹配", bar: "bg-amber-500", label: "text-amber-700", delta: "text-amber-600" },
    { key: "保", meaning: "兜得住 · 保底", bar: "bg-emerald-500", label: "text-emerald-700", delta: "text-emerald-600" },
  ];

  // 每个选项相对“你”的位次差：高你 = 录取线更靠前（要往上够），低你 = 你已越过（有富余）。
  function delta(R: number): string {
    const d = R - V;
    if (d === 0) return "与你持平";
    return d < 0 ? `↑ 高你 ${(-d).toLocaleString()} 位` : `↓ 低你 ${d.toLocaleString()} 位`;
  }

  const seg = (active: boolean) =>
    `rounded-md px-3 py-1 text-sm font-medium transition ${
      active ? "bg-white text-slate-900 shadow-sm" : "text-slate-500 hover:text-slate-700"
    }`;

  return (
    <div>
      {/* 控制台 + 位次主角 */}
      <div class="rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
        <div class="flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between">
          {/* 输入 */}
          <div class="space-y-3">
            <div class="flex flex-wrap items-center gap-2">
              <div class="inline-flex rounded-lg bg-slate-100 p-0.5">
                {(["物理", "历史"] as Track[]).map((t) => (
                  <button type="button" onClick={() => setTrack(t)} class={seg(track === t)}>
                    {t}类
                  </button>
                ))}
              </div>
              {track === "物理" && (
                <div class="inline-flex rounded-lg bg-slate-100 p-0.5">
                  <button type="button" onClick={() => setMode("score")} class={seg(mode === "score")}>
                    分数
                  </button>
                  <button type="button" onClick={() => setMode("rank")} class={seg(mode === "rank")}>
                    位次
                  </button>
                </div>
              )}
              <input
                type="number"
                inputMode="numeric"
                value={val}
                onInput={(e) => setVal((e.target as HTMLInputElement).value)}
                placeholder={effectiveMode === "score" ? "输入分数" : "输入位次"}
                class="w-28 rounded-lg border border-slate-300 px-3 py-1.5 text-sm focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
              />
            </div>
            <div class="flex flex-wrap items-center gap-1.5">
              <span class="text-xs text-slate-400">再选科目</span>
              {RESELECT.map((s) => (
                <button
                  type="button"
                  onClick={() => setSel((p) => ({ ...p, [s]: !p[s] }))}
                  class={`rounded-full px-2.5 py-0.5 text-xs font-medium transition ${
                    sel[s] ? "bg-slate-800 text-white" : "border border-slate-300 text-slate-500 hover:border-slate-400"
                  }`}
                >
                  {s}
                </button>
              ))}
            </div>
          </div>

          {/* 你的位次：本屏主角 */}
          <div class="shrink-0 sm:text-right">
            {V > 0 ? (
              <>
                <div class="text-xs font-medium tracking-wide text-slate-400">你的全省位次</div>
                <div class="mt-0.5 text-4xl font-bold tabular-nums tracking-tight text-slate-900 sm:text-5xl">
                  {V.toLocaleString()}
                </div>
                <div class="mt-1 text-xs text-slate-500">{track}类 · 等效到 2026 · 越小越靠前</div>
              </>
            ) : (
              <div class="text-sm text-slate-400">
                输入{effectiveMode === "score" ? "分数" : "位次"}，立即定位你的冲稳保
              </div>
            )}
          </div>
        </div>
        {track === "历史" && (
          <p class="mt-3 text-xs text-amber-700">历史类暂缺 2026 一分一段，请直接输入位次。</p>
        )}
      </div>

      {/* 结果 */}
      {loading && <p class="mt-6 text-sm text-slate-500">加载定位数据…</p>}
      {!loading && V > 0 && (
        <div class="mt-6 grid gap-4 lg:grid-cols-3">
          {cfg.map(({ key, meaning, bar, label, delta: deltaCls }) => {
            const list = buckets[key];
            return (
              <div class="overflow-hidden rounded-xl border border-slate-200 bg-white">
                <div class={`h-1 ${bar}`} />
                <div class="flex items-baseline justify-between px-3 pt-2.5">
                  <div class="flex items-baseline gap-1.5">
                    <span class={`text-base font-bold ${label}`}>{key}</span>
                    <span class="text-xs text-slate-400">{meaning}</span>
                  </div>
                  <span class="text-xs tabular-nums text-slate-400">
                    {list.length}
                    {list.length > CAP ? ` · 前 ${CAP}` : ""} 个
                  </span>
                </div>
                <div class="px-3 pb-1.5 pt-1 text-right text-[10px] tracking-wide text-slate-400">
                  录取位次 · 与你差距
                </div>
                <ul class="divide-y divide-slate-100 border-t border-slate-100">
                  {list.slice(0, CAP).map((e) => (
                    <li>
                      <a href={`/yuanxiao/${e.sc}/#z-${e.mk}`} class="block px-3 py-2 hover:bg-slate-50">
                        <div class="flex items-start justify-between gap-2">
                          <div class="min-w-0">
                            <div class="truncate text-sm font-medium text-slate-900">{e.sn}</div>
                            <div class="truncate text-xs text-slate-500">{e.mn}</div>
                          </div>
                          <div class="shrink-0 text-right leading-tight">
                            <div class="text-sm font-semibold tabular-nums text-slate-800">
                              {e.r.toLocaleString()}
                            </div>
                            <div class={`text-[11px] tabular-nums ${deltaCls}`}>{delta(e.r)}</div>
                          </div>
                        </div>
                        <div class="mt-1 flex flex-wrap gap-x-2 text-[11px] text-slate-400">
                          <span>计划 {e.pl || "—"}</span>
                          <span>选科 {e.sk || "不限"}</span>
                          {e.gs > 1 && <span>组内 {e.gs} 专业 · 服从可调剂</span>}
                        </div>
                      </a>
                    </li>
                  ))}
                  {list.length === 0 && (
                    <li class="px-3 py-3 text-xs text-slate-300">这一档暂无可填</li>
                  )}
                </ul>
              </div>
            );
          })}
        </div>
      )}
      {!loading && V <= 0 && (
        <div class="mt-6 rounded-xl border border-dashed border-slate-300 bg-white px-4 py-10 text-center">
          <p class="text-sm text-slate-500">
            输入分数或位次，按等效位次给出 <span class="font-medium text-rose-600">冲</span> /{" "}
            <span class="font-medium text-amber-600">稳</span> /{" "}
            <span class="font-medium text-emerald-600">保</span> 的可填报院校专业组。
          </p>
        </div>
      )}
    </div>
  );
}
