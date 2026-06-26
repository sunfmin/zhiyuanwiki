import { useEffect, useMemo, useState } from "preact/hooks";

type TR = { year: number; minScore: number; maxScore: number; minRank: number; maxRank: number };
type S = {
  code: string;
  name: string;
  leafCount: number;
  wuli?: TR; // 物理类
  lishi?: TR; // 历史类
  is985?: boolean;
  is211?: boolean;
  isShuangYiLiu?: boolean;
};

type Track = "物理" | "历史";
const LS_KEY = "yuanxiao.sortTrack";
const fmt = (n?: number) => (n ? n.toLocaleString("en-US") : "");
const rangeOf = (s: S, t: Track) => (t === "物理" ? s.wuli : s.lishi);

// 院校层次角标：985 ⊃ 211 ⊃ 双一流，取最高一档显示（已隐含更低档）。
function tier(s: S): { label: string; cls: string } | null {
  if (s.is985) return { label: "985", cls: "bg-rose-100 text-rose-700 ring-rose-200" };
  if (s.is211) return { label: "211", cls: "bg-indigo-100 text-indigo-700 ring-indigo-200" };
  if (s.isShuangYiLiu) return { label: "双一流", cls: "bg-amber-100 text-amber-700 ring-amber-200" };
  return null;
}

/** 一所院校的某科类录取线区间行；该科类无数据则不渲染。 */
function TrackLine({ track, r, active }: { track: Track; r?: TR; active: boolean }) {
  if (!r) return null;
  const labelCls = active ? "font-semibold text-slate-600" : "text-slate-400";
  return (
    <div class="flex flex-wrap items-baseline gap-x-2.5 gap-y-0.5">
      <span class={`w-7 shrink-0 ${labelCls}`}>{track}</span>
      <span>
        分 <span class="font-medium text-slate-700">{r.minScore}–{r.maxScore}</span>
      </span>
      {r.maxRank ? (
        <span>
          位次 <span class="font-medium text-slate-700">{fmt(r.minRank)}–{fmt(r.maxRank)}</span>
        </span>
      ) : null}
      <span class="text-slate-300">{r.year}</span>
    </div>
  );
}

/** 按院校名/代码站内搜索 + 按物理/历史录取分排序，每校物理、历史录取线区间分开列。 */
export default function SchoolSearch({ schools }: { schools: S[] }) {
  const [q, setQ] = useState("");
  const [sortTrack, setSortTrack] = useState<Track>("物理");

  // 记住上次选的排序科类。
  const [ready, setReady] = useState(false);
  useEffect(() => {
    try {
      const v = localStorage.getItem(LS_KEY);
      if (v === "物理" || v === "历史") setSortTrack(v);
    } catch {
      /* 忽略 */
    }
    setReady(true);
  }, []);
  useEffect(() => {
    if (!ready) return;
    try {
      localStorage.setItem(LS_KEY, sortTrack);
    } catch {
      /* 忽略 */
    }
  }, [ready, sortTrack]);

  const list = useMemo(() => {
    const t = q.trim();
    const filtered = t ? schools.filter((s) => s.name.includes(t) || s.code.includes(t)) : schools.slice();
    // 按所选科类的"顶尖专业录取分"降序；无该科类数据的院校沉底，再按专业数、校名兜底。
    const key = (s: S) => rangeOf(s, sortTrack)?.maxScore ?? -1;
    return filtered.sort(
      (a, b) => key(b) - key(a) || b.leafCount - a.leafCount || a.name.localeCompare(b.name, "zh"),
    );
  }, [q, schools, sortTrack]);

  const tabCls = (t: Track) =>
    `rounded-md px-3 py-1 text-sm font-medium ${
      sortTrack === t ? "bg-slate-900 text-white" : "bg-slate-100 text-slate-600 hover:bg-slate-200"
    }`;

  return (
    <div>
      <div class="flex flex-wrap items-center gap-3">
        <input
          type="search"
          value={q}
          onInput={(e) => setQ((e.target as HTMLInputElement).value)}
          placeholder="按院校名或代码搜索，如 哈尔滨工业大学"
          class="w-full max-w-md rounded-md border border-slate-300 px-3 py-2"
        />
        <div class="flex items-center gap-1.5">
          <span class="text-sm text-slate-500">排序：</span>
          {(["物理", "历史"] as Track[]).map((t) => (
            <button type="button" onClick={() => setSortTrack(t)} class={tabCls(t)}>
              {t}类
            </button>
          ))}
        </div>
      </div>
      <p class="mt-2 text-sm text-slate-500">
        {q.trim() ? `匹配 ${list.length} 所` : `共 ${schools.length} 所院校`} · 按{sortTrack}类顶尖专业录取分从高到低
      </p>

      <ul class="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {list.map((s) => (
          <li>
            <a
              href={`/yuanxiao/${s.code}/`}
              class="block rounded-lg border border-slate-200 bg-white px-3 py-2 hover:border-slate-400"
            >
              <div class="flex items-baseline justify-between gap-2">
                <span class="min-w-0">
                  <span class="font-medium">{s.name}</span>
                  {tier(s) && (
                    <span
                      class={`ml-1.5 rounded px-1 py-0.5 text-[10px] font-semibold ring-1 ring-inset align-middle ${tier(s)!.cls}`}
                    >
                      {tier(s)!.label}
                    </span>
                  )}
                </span>
                <span class="shrink-0 text-xs text-slate-400">{s.code} · {s.leafCount} 专业</span>
              </div>
              <div class="mt-1 space-y-0.5 text-xs text-slate-500">
                <TrackLine track="物理" r={s.wuli} active={sortTrack === "物理"} />
                <TrackLine track="历史" r={s.lishi} active={sortTrack === "历史"} />
                {!s.wuli && !s.lishi && <div class="text-slate-300">暂无录取线数据</div>}
              </div>
            </a>
          </li>
        ))}
        {list.length === 0 && <li class="text-sm text-slate-400">没有匹配的院校</li>}
      </ul>
    </div>
  );
}
