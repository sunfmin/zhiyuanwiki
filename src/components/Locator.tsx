import { useEffect, useMemo, useRef, useState } from "preact/hooks";
import { scoreToRank, type YiFenYiDuan } from "../lib/fenduan";
import { classify, selKeAllows, type Bucket, type LocEntry } from "../lib/dingwei";

type Track = "物理" | "历史";
const RESELECT = ["化学", "生物", "政治", "地理"] as const;
const CAP = 60;

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

  const cfg: { key: Bucket; label: string; cls: string }[] = [
    { key: "冲", label: "冲", cls: "border-rose-300 bg-rose-50" },
    { key: "稳", label: "稳", cls: "border-amber-300 bg-amber-50" },
    { key: "保", label: "保", cls: "border-emerald-300 bg-emerald-50" },
  ];

  return (
    <div>
      <div class="rounded-xl border border-slate-200 bg-white p-5">
        {/* 科类 */}
        <div class="flex flex-wrap items-center gap-4">
          <div class="flex gap-2">
            {(["物理", "历史"] as Track[]).map((t) => (
              <button
                type="button"
                onClick={() => setTrack(t)}
                class={`rounded-md px-3 py-1.5 text-sm font-medium ${
                  track === t ? "bg-slate-900 text-white" : "bg-slate-100 text-slate-600"
                }`}
              >
                {t}类
              </button>
            ))}
          </div>

          {track === "物理" && (
            <div class="flex gap-2 text-sm">
              <label class="flex items-center gap-1">
                <input type="radio" checked={mode === "score"} onChange={() => setMode("score")} /> 分数
              </label>
              <label class="flex items-center gap-1">
                <input type="radio" checked={mode === "rank"} onChange={() => setMode("rank")} /> 位次
              </label>
            </div>
          )}

          <input
            type="number"
            inputMode="numeric"
            value={val}
            onInput={(e) => setVal((e.target as HTMLInputElement).value)}
            placeholder={effectiveMode === "score" ? "输入分数" : "输入位次"}
            class="w-36 rounded-md border border-slate-300 px-3 py-2"
          />
          {V > 0 && (
            <span class="text-sm text-slate-600">
              你的位次约 <strong class="text-slate-900">{V.toLocaleString()}</strong>
            </span>
          )}
        </div>

        {/* 再选科目 */}
        <div class="mt-3 flex flex-wrap items-center gap-3 text-sm">
          <span class="text-slate-500">再选：</span>
          {RESELECT.map((s) => (
            <label class="flex items-center gap-1">
              <input
                type="checkbox"
                checked={sel[s]}
                onChange={() => setSel((p) => ({ ...p, [s]: !p[s] }))}
              />
              {s}
            </label>
          ))}
        </div>
        {track === "历史" && (
          <p class="mt-2 text-xs text-amber-700">历史类暂缺 2026 一分一段，请直接输入位次。</p>
        )}
      </div>

      {/* 结果 */}
      {loading && <p class="mt-6 text-sm text-slate-500">加载定位数据…</p>}
      {!loading && V > 0 && (
        <div class="mt-6 grid gap-4 lg:grid-cols-3">
          {cfg.map(({ key, label, cls }) => {
            const list = buckets[key];
            return (
              <div class={`rounded-xl border ${cls} p-3`}>
                <div class="flex items-baseline justify-between">
                  <h3 class="text-base font-bold">{label}</h3>
                  <span class="text-xs text-slate-500">
                    {list.length > CAP ? `${list.length} 个 · 显示前 ${CAP}` : `${list.length} 个`}
                  </span>
                </div>
                <ul class="mt-2 space-y-2">
                  {list.slice(0, CAP).map((e) => (
                    <li class="rounded-lg bg-white/70 p-2 text-sm">
                      <a
                        href={`/yuanxiao/${e.sc}/zhuanye/${e.mk}/`}
                        class="font-medium text-slate-900 hover:underline"
                      >
                        {e.sn} · {e.mn}
                      </a>
                      <div class="mt-0.5 text-xs text-slate-500">
                        {e.gn}（选科：{e.sk || "不限"}） · 等效位次 {e.r.toLocaleString()} · 计划 {e.pl || "—"}
                      </div>
                      {e.gs > 1 && (
                        <div class="text-xs text-slate-400">服从调剂可能被调到组内其余 {e.gs - 1} 个专业</div>
                      )}
                    </li>
                  ))}
                  {list.length === 0 && <li class="text-xs text-slate-400">无</li>}
                </ul>
              </div>
            );
          })}
        </div>
      )}
      {!loading && V <= 0 && (
        <p class="mt-6 text-sm text-slate-500">输入分数或位次后，按等效位次给出冲 / 稳 / 保的可填报院校专业组。</p>
      )}
    </div>
  );
}
