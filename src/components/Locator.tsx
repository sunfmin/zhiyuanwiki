import { useEffect, useMemo, useRef, useState } from "preact/hooks";
import { scoreToRank, rankToScore, type YiFenYiDuan } from "../lib/fenduan";
import {
  bucketize,
  assembleColumns,
  reachColor,
  selKeAllows,
  selKeAllowsZJ,
  type MainTier,
  type LocEntry,
  type ReachLevel,
} from "../lib/dingwei";
import { provinceConfig, trackSlugOf } from "../lib/provinces";
import {
  matchesFilters,
  emptyFilters,
  anyActive,
  CATEGORIES,
  CATEGORY_LABEL,
  LEVELS,
  OWNERSHIPS,
  CITY_TIERS,
  type Filters,
  type SchoolMetaMap,
} from "../lib/filters";

const PRIMARY_RESELECT = ["化学", "生物", "政治", "地理"]; // 黑龙江：首选物理/历史外的再选
const ZJ_SUBJECTS = ["物理", "化学", "生物", "政治", "历史", "地理", "技术"]; // 浙江 7选3
const PICK3_SUBJECTS = ["物理", "化学", "生物", "政治", "历史", "地理"]; // 北京/上海/海南/山东 6选3（无技术）
// 远档预览的说明文案。承重的分列/截断/补齐规则已在 dingwei.assembleColumns（见 ADR-0010），这里只剩 UI copy。
const FAR_NOTE: Record<"够不着" | "过保", string> = {
  够不着: "比冲更难，基本搏不到",
  过保: "比保更易，白白浪费位次",
};
// 把握配色档 → 文字色类（承重阈值在 dingwei.reachColor，这里只剩 view 映射）。
const REACH_CLS: Record<ReachLevel, string> = {
  easy: "text-emerald-600",
  mid: "text-amber-600",
  hard: "text-rose-600",
};

export default function Locator({ prov, table }: { prov: string; table: YiFenYiDuan }) {
  const cfg = provinceConfig(prov);
  const multiTrack = cfg.tracks.length > 1;
  // wenli：老高考（新疆 理科/文科）——无选科，仅理科/文科科类切换；选科判定恒真、隐藏选科区。
  const wenli = cfg.subjectMode === "wenli";
  // pick3：综合「选3」模型——浙江 7选3（含技术）与北京/上海/海南/山东 6选3（无技术）共用同一套
  // 选科判定（selKeAllowsZJ）与「最多 3 科」逻辑，仅候选科目表不同。
  const pick3 = cfg.subjectMode === "pick3of7" || cfg.subjectMode === "pick3of6";
  const pick3Subjects = cfg.subjectMode === "pick3of7" ? ZJ_SUBJECTS : PICK3_SUBJECTS;
  const matchSelKe = wenli ? () => true : pick3 ? selKeAllowsZJ : selKeAllows;
  const unit = cfg.fillModel === "group" ? "院校专业组" : "院校×专业";
  const LS_KEY = `dingwei.input.${prov}`;

  const [track, setTrack] = useState<string>(cfg.tracks[0].name);
  const [mode, setMode] = useState<"score" | "rank">("score");
  const [val, setVal] = useState("");
  const [sel, setSel] = useState<Record<string, boolean>>(
    pick3 ? { 物理: true, 化学: true, 生物: true } : { 化学: true, 生物: true, 政治: false, 地理: false },
  );
  const [filters, setFilters] = useState<Filters>(emptyFilters);
  const [panelOpen, setPanelOpen] = useState(false);
  const [activeTier, setActiveTier] = useState<MainTier>("冲");
  const [expanded, setExpanded] = useState<Record<MainTier, boolean>>({ 冲: false, 稳: false, 保: false });

  const [entries, setEntries] = useState<LocEntry[]>([]);
  const [meta, setMeta] = useState<SchoolMetaMap>({});
  const [loading, setLoading] = useState(false);
  const cache = useRef<Record<string, LocEntry[]>>({});

  // 当前科类是否有一分一段表（=能按分数输入）；否则强制按位次（如黑龙江历史类）。
  const canScore = track === cfg.fenduanTrack;
  const effectiveMode: "score" | "rank" = canScore ? mode : "rank";

  // 挂载后恢复上次输入（localStorage，按省命名空间）；ready 之前不回写。
  const [ready, setReady] = useState(false);
  useEffect(() => {
    try {
      const raw = localStorage.getItem(LS_KEY);
      if (raw) {
        const s = JSON.parse(raw) as Partial<{
          track: string;
          mode: "score" | "rank";
          val: string;
          sel: Record<string, boolean>;
          filters: Partial<Filters>;
        }>;
        if (s.track && cfg.tracks.some((t) => t.name === s.track)) setTrack(s.track);
        if (s.mode === "score" || s.mode === "rank") setMode(s.mode);
        if (typeof s.val === "string") setVal(s.val);
        if (s.sel && typeof s.sel === "object") setSel(s.sel);
        if (s.filters && typeof s.filters === "object") setFilters((p) => ({ ...p, ...s.filters }));
      }
    } catch {
      /* 隐私模式 / 损坏数据忽略 */
    }
    setReady(true);
  }, []);

  useEffect(() => {
    if (!ready) return;
    try {
      localStorage.setItem(LS_KEY, JSON.stringify({ track, mode, val, sel, filters }));
    } catch {
      /* 忽略 */
    }
  }, [ready, track, mode, val, sel, filters]);

  useEffect(() => {
    fetch(`/data/${prov}/school-meta.json`)
      .then((r) => r.json())
      .then((d: SchoolMetaMap) => setMeta(d))
      .catch(() => {
        /* 拿不到则院校级过滤不生效，专业级仍可用 */
      });
  }, [prov]);

  useEffect(() => {
    const slug = trackSlugOf(cfg, track);
    const file = `/data/${prov}/locator-${slug}.json`;
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
  }, [prov, track]);

  // 已选科目集合：老文理(wenli)无选科→空集（matchSelKe 恒真，集合不参与判定）；
  // 黑龙江=首选+再选；浙江=7选3 所选。
  const chosen = useMemo(() => {
    if (wenli) return new Set<string>();
    if (pick3) {
      const s = new Set<string>();
      for (const k of pick3Subjects) if (sel[k]) s.add(k);
      return s;
    }
    const s = new Set<string>([track]);
    for (const k of PRIMARY_RESELECT) if (sel[k]) s.add(k);
    return s;
  }, [track, sel, pick3, wenli]);

  const V = useMemo(() => {
    const n = parseInt(val, 10);
    if (Number.isNaN(n) || n <= 0) return 0;
    if (effectiveMode === "rank") return n;
    return scoreToRank(table, n) ?? 0;
  }, [val, effectiveMode, table]);

  useEffect(() => {
    if (V > 0) {
      try {
        localStorage.setItem(`myRank.${prov}`, String(V));
        localStorage.setItem(`myTrack.${prov}`, track);
      } catch {
        /* 隐私模式等忽略 */
      }
    }
  }, [V, track, prov]);

  const provinceOpts = useMemo(() => distinctSorted(Object.values(meta).map((m) => m.p)), [meta]);
  const kindOpts = useMemo(() => distinctSorted(Object.values(meta).map((m) => m.k)), [meta]);

  const activeFilters = anyActive(filters);

  const buckets = useMemo(() => {
    // 先按选科 + 用户筛选过滤，再交给纯函数 bucketize 按把握比值分档（每档最贴你水平在前，不凑数）。
    const eligible: LocEntry[] = [];
    for (const e of entries) {
      if (e.r <= 0) continue;
      if (!matchSelKe(e.sk, chosen)) continue;
      if (!matchesFilters(e, meta, filters)) continue;
      eligible.push(e);
    }
    return bucketize(V, eligible);
  }, [entries, V, chosen, meta, filters, matchSelKe]);

  // 三列装配（截断 + 远档补齐到约 100）全在纯函数里，承重规则不再内联于 render。
  const columns = useMemo(() => assembleColumns(buckets), [buckets]);

  const cfgT: { key: MainTier; meaning: string; bar: string; label: string; tint: string }[] = [
    { key: "冲", meaning: "够一够 · 偏难", bar: "bg-rose-500", label: "text-rose-700", tint: "bg-rose-50 ring-rose-200" },
    { key: "稳", meaning: "较稳妥 · 匹配", bar: "bg-amber-500", label: "text-amber-700", tint: "bg-amber-50 ring-amber-200" },
    { key: "保", meaning: "兜得住 · 保底", bar: "bg-emerald-500", label: "text-emerald-700", tint: "bg-emerald-50 ring-emerald-200" },
  ];

  // 等效分：位次仍是基准，分数只作"感知"辅助，仅在当前科类有一分一段表时给出，并标"约"。
  const myScore = canScore && V > 0 ? rankToScore(table, V) : null;
  const scoreOf = (R: number) => (canScore ? rankToScore(table, R) : null);

  function delta(R: number): string {
    const d = R - V;
    if (d === 0) return "与你持平";
    return d < 0 ? `↑ 高你 ${(-d).toLocaleString()} 位` : `↓ 低你 ${d.toLocaleString()} 位`;
  }

  // 分数差（约）：录取线高你几分 / 低你几分。
  function scoreDelta(R: number): string | null {
    if (myScore == null) return null;
    const rs = scoreOf(R);
    if (rs == null) return null;
    const d = rs - myScore;
    if (d === 0) return "约 持平";
    return d > 0 ? `约 高你 ${d} 分` : `约 低你 ${-d} 分`;
  }

  // 按把握给差距着色：稳/保→绿、较易冲→琥珀、偏难冲/够不着→红。阈值在 dingwei.reachColor（与 classify 同源）。
  const reachTint = (R: number) => REACH_CLS[reachColor(V, R)];

  // 单条候选行。主列与远档预览复用；muted=远档预览（置灰）。
  const renderRow = (e: LocEntry, muted = false) => {
    const m = meta[e.sc];
    const lv = topLevel(m?.lv);
    const city = m?.c?.replace(/[市]$/, "");
    const rs = scoreOf(e.r);
    const sd = scoreDelta(e.r);
    return (
      <li key={`${e.sc}-${e.mk}`}>
        <a
          href={`/${prov}/yuanxiao/${e.sc}/#z-${e.mk}`}
          class={`block px-3 py-2 hover:bg-slate-50 ${muted ? "opacity-60 grayscale" : ""}`}
        >
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <div class="flex items-center gap-1.5">
                <span class="truncate text-sm font-medium text-slate-900">{e.sn}</span>
                {lv && (
                  <span class={`shrink-0 rounded px-1 py-px text-[10px] font-medium ${levelCls(lv)}`}>{lv}</span>
                )}
              </div>
              <div class="truncate text-xs text-slate-500">{e.mn}</div>
            </div>
            <div class="shrink-0 text-right leading-tight">
              {rs != null ? (
                <>
                  <div class="text-sm font-semibold tabular-nums text-slate-800">约 {rs} 分</div>
                  {sd && <div class={`text-[11px] tabular-nums ${reachTint(e.r)}`}>{sd}</div>}
                  <div class="text-[10px] tabular-nums text-slate-400">
                    位次 {e.r.toLocaleString()} · {delta(e.r)}
                  </div>
                </>
              ) : (
                <>
                  <div class="text-sm font-semibold tabular-nums text-slate-800">{e.r.toLocaleString()}</div>
                  <div class={`text-[11px] tabular-nums ${reachTint(e.r)}`}>{delta(e.r)}</div>
                </>
              )}
            </div>
          </div>
          <div class="mt-1 flex flex-wrap gap-x-2 text-[11px] text-slate-400">
            {city && <span class="text-slate-500">{city}</span>}
            {m?.k && <span>{m.k}</span>}
            <span>计划 {e.pl || "—"}</span>
            <span>选科 {e.sk || "不限"}</span>
            {cfg.fillModel === "group" && (e.gs ?? 0) > 1 && <span>组内 {e.gs} 专业 · 服从可调剂</span>}
            {m?.o === "民办" && <span class="text-amber-600">民办</span>}
            {e.cw && <span class="text-violet-500">中外合作</span>}
          </div>
        </a>
      </li>
    );
  };

  const seg = (active: boolean) =>
    `rounded-md px-3 py-1 text-sm font-medium transition ${
      active ? "bg-white text-slate-900 shadow-sm" : "text-slate-500 hover:text-slate-700"
    }`;

  function toggle(key: "provinces" | "levels" | "ownership" | "kinds" | "cityTiers" | "categories", v: string) {
    setFilters((p) => {
      const arr = p[key];
      return { ...p, [key]: arr.includes(v) ? arr.filter((x) => x !== v) : [...arr, v] };
    });
  }
  function clearAll() {
    setFilters(emptyFilters());
  }

  // 浙江 7选3：最多选 3 科。
  function toggleSubject(s: string) {
    setSel((p) => {
      const on = !!p[s];
      if (!on && pick3 && pick3Subjects.filter((k) => p[k]).length >= 3) return p; // 已满 3 科
      return { ...p, [s]: !on };
    });
  }

  // 抽屉内（高级/体积大）维度的已选 chip：抽屉收起时也常显，便于一眼看到并移除被折叠的筛选。
  // 快捷区维度（大类/层次/城市/关键词）的选中态在常显控件上已可见，不再重复成 chip。
  const advChips: { id: string; text: string; remove: () => void }[] = [];
  for (const o of filters.ownership) advChips.push({ id: `o-${o}`, text: o, remove: () => toggle("ownership", o) });
  for (const k of filters.kinds) advChips.push({ id: `k-${k}`, text: k, remove: () => toggle("kinds", k) });
  for (const p of filters.provinces) advChips.push({ id: `p-${p}`, text: p, remove: () => toggle("provinces", p) });
  if (filters.minPlan > 0)
    advChips.push({ id: "mp", text: `计划 ≥ ${filters.minPlan}`, remove: () => setFilters((p) => ({ ...p, minPlan: 0 })) });
  if (cfg.fillModel === "group" && filters.maxGroupSize > 0)
    advChips.push({ id: "mg", text: `组内 ≤ ${filters.maxGroupSize}`, remove: () => setFilters((p) => ({ ...p, maxGroupSize: 0 })) });
  if (filters.hideCoopHighFee)
    advChips.push({ id: "hf", text: "隐藏中外 / 高收费", remove: () => setFilters((p) => ({ ...p, hideCoopHighFee: false })) });

  return (
    <div>
      <div class="rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
        {/* 数据基准年：明确告知位次换算用的是哪一年的官方一分一段（取自实际装载的表，非配置常量）。 */}
        {canScore && (
          <div class="mb-4 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs">
            <span class="inline-flex items-center rounded-md bg-emerald-50 px-2 py-0.5 font-semibold text-emerald-700 ring-1 ring-emerald-600/20">
              {table.year} 年一分一段
            </span>
            <span class="text-slate-500">
              位次换算基于{table.province}
              {table.year}年{table.track}类官方一分一段表（{table.entries.length} 个分数段）
            </span>
          </div>
        )}
        <div class="flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between">
          <div class="space-y-3">
            <div class="flex flex-wrap items-center gap-2">
              {multiTrack && (
                <div class="inline-flex rounded-lg bg-slate-100 p-0.5">
                  {cfg.tracks.map((t) => (
                    <button type="button" onClick={() => setTrack(t.name)} class={seg(track === t.name)}>
                      {t.name}
                      {wenli ? "" : "类"}
                    </button>
                  ))}
                </div>
              )}
              {canScore && (
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
            {/* 老文理(wenli)无选科，隐藏整个选科区；其余省份显再选/选考科目 chip。 */}
            {!wenli && (
              <div class="flex flex-wrap items-center gap-1.5">
                <span class="text-xs text-slate-400">
                  {pick3 ? `选考科目（${pick3Subjects.length}选3）` : "再选科目"}
                </span>
                {(pick3 ? pick3Subjects : PRIMARY_RESELECT).map((s) => (
                  <button
                    type="button"
                    onClick={() => toggleSubject(s)}
                    class={`rounded-full px-2.5 py-0.5 text-xs font-medium transition ${
                      sel[s] ? "bg-slate-800 text-white" : "border border-slate-300 text-slate-500 hover:border-slate-400"
                    }`}
                  >
                    {s}
                  </button>
                ))}
              </div>
            )}
          </div>

          <div class="shrink-0 sm:text-right">
            {V > 0 ? (
              <>
                <div class="text-xs font-medium tracking-wide text-slate-400">你的全省位次</div>
                <div class="mt-0.5 text-4xl font-bold tabular-nums tracking-tight text-slate-900 sm:text-5xl">
                  {V.toLocaleString()}
                </div>
                <div class="mt-1 text-xs text-slate-500">
                  {track}
                  {wenli ? "" : "类"} · 等效到 {cfg.fenduanYear} · 越小越靠前
                </div>
              </>
            ) : (
              <div class="text-sm text-slate-400">
                输入{effectiveMode === "score" ? "分数" : "位次"}，立即定位你的冲稳保
              </div>
            )}
          </div>
        </div>
        {multiTrack && !canScore && !wenli && (
          <p class="mt-3 text-xs text-amber-700">{track}类暂缺 {cfg.fenduanYear} 一分一段，请直接输入位次。</p>
        )}
        {/* 老文理：文科按位次定位（分数↔位次换算仅理科接入），用中性提示替代「暂缺」误导文案。 */}
        {wenli && !canScore && (
          <p class="mt-3 text-xs text-slate-500">{track}按位次定位（分数换算仅理科）。</p>
        )}
      </div>

      {V > 0 && (
        <div class="mt-4 rounded-xl border border-slate-200 bg-white">
          {/* 常显快捷筛选：最常用的几项无需展开即可用——选什么（大类 / 关键词）、够多好（层次）、大致在哪（城市层级）。 */}
          <div class="space-y-4 px-4 py-4">
            <ChipRow
              label="专业大类"
              options={CATEGORIES.map((c) => c.code)}
              selected={filters.categories}
              labelOf={(c) => CATEGORY_LABEL[c] || c}
              onToggle={(v) => toggle("categories", v)}
            />
            <div class="flex flex-wrap gap-x-8 gap-y-4">
              <ChipRow label="院校层次" options={[...LEVELS]} selected={filters.levels} onToggle={(v) => toggle("levels", v)} />
              <ChipRow label="城市层级" options={[...CITY_TIERS]} selected={filters.cityTiers} onToggle={(v) => toggle("cityTiers", v)} />
            </div>
            {/* 院校 / 专业两个关键词框并排：框内空格分隔=任一匹配(OR)，两框之间 AND，于是「某校的某专业」可精确命中。 */}
            <div class="flex flex-wrap gap-x-6 gap-y-4">
              <KeywordField
                label="院校关键词"
                value={filters.schoolKeyword}
                onInput={(v) => setFilters((p) => ({ ...p, schoolKeyword: v }))}
                placeholder="空格分隔=任一匹配，如 浙江大学 师范"
              />
              <KeywordField
                label="专业关键词"
                value={filters.majorKeyword}
                onInput={(v) => setFilters((p) => ({ ...p, majorKeyword: v }))}
                placeholder="空格分隔=任一匹配，如 计算机 软件"
              />
            </div>
          </div>

          {/* 控制条：更多筛选开关 + 抽屉内已选 chip（收起时也常显，避免「看不见的筛选」）+ 清除全部。 */}
          <div class="flex flex-wrap items-center gap-2 border-t border-slate-100 px-4 py-2.5">
            <button
              type="button"
              onClick={() => setPanelOpen((o) => !o)}
              aria-expanded={panelOpen}
              class={`inline-flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-sm font-medium transition ${
                advChips.length > 0
                  ? "border-slate-800 text-slate-800"
                  : "border-slate-300 text-slate-600 hover:border-slate-400"
              }`}
            >
              <span>更多筛选</span>
              {advChips.length > 0 && (
                <span class="rounded-full bg-slate-800 px-1.5 text-xs tabular-nums text-white">{advChips.length}</span>
              )}
              <span class="text-xs text-slate-400">{panelOpen ? "▴" : "▾"}</span>
            </button>

            {!panelOpen &&
              advChips.map((c) => (
                <button
                  type="button"
                  key={c.id}
                  onClick={c.remove}
                  class="inline-flex max-w-full items-center gap-1 rounded-full bg-slate-100 px-2.5 py-1 text-left text-xs text-slate-700 hover:bg-slate-200"
                >
                  <span class="min-w-0 break-words">{c.text}</span>
                  <span class="shrink-0 text-slate-400">✕</span>
                </button>
              ))}
            {activeFilters && (
              <button type="button" onClick={clearAll} class="ml-auto text-xs text-slate-400 hover:text-slate-700">
                清除全部
              </button>
            )}
          </div>

          {/* 抽屉：不常用 / 选项体积大的维度，浅底色区分于上方常显的主筛选。 */}
          {panelOpen && (
            <div class="space-y-4 rounded-b-xl border-t border-slate-100 bg-slate-50/70 px-4 py-4">
              <ChipRow label="办学性质" options={[...OWNERSHIPS]} selected={filters.ownership} onToggle={(v) => toggle("ownership", v)} />
              {kindOpts.length > 0 && (
                <ChipRow label="学校类别" options={kindOpts} selected={filters.kinds} onToggle={(v) => toggle("kinds", v)} />
              )}
              {provinceOpts.length > 0 && (
                <ChipRow label="省份（院校所在地）" options={provinceOpts} selected={filters.provinces} onToggle={(v) => toggle("provinces", v)} />
              )}
              <div class="flex flex-wrap items-end gap-x-6 gap-y-3">
                <NumField
                  label="计划人数下限"
                  value={filters.minPlan}
                  onChange={(n) => setFilters((p) => ({ ...p, minPlan: n }))}
                />
                {cfg.fillModel === "group" && (
                  <NumField
                    label="组内专业数上限"
                    value={filters.maxGroupSize}
                    onChange={(n) => setFilters((p) => ({ ...p, maxGroupSize: n }))}
                  />
                )}
                <label class="flex cursor-pointer items-center gap-2 text-sm text-slate-700">
                  <input
                    type="checkbox"
                    checked={filters.hideCoopHighFee}
                    onChange={(e) => setFilters((p) => ({ ...p, hideCoopHighFee: (e.target as HTMLInputElement).checked }))}
                    class="h-4 w-4 rounded border-slate-300"
                  />
                  隐藏中外合作及高收费（≥2万/年）
                </label>
              </div>
            </div>
          )}
        </div>
      )}

      {loading && <p class="mt-6 text-sm text-slate-500">加载定位数据…</p>}
      {!loading && V > 0 && (
        <>
          <div class="sticky top-0 z-20 mt-4 bg-slate-50/95 py-2 backdrop-blur lg:hidden">
            <div class="flex gap-1 rounded-xl border border-slate-200 bg-white p-1 shadow-sm">
              {cfgT.map(({ key, label, tint }) => {
                const on = key === activeTier;
                return (
                  <button
                    type="button"
                    onClick={() => setActiveTier(key)}
                    aria-pressed={on}
                    class={`flex-1 rounded-lg py-1.5 text-center transition ${on ? `${tint} ring-1` : "hover:bg-slate-50"}`}
                  >
                    <span class={`text-sm font-bold ${on ? label : "text-slate-500"}`}>{key}</span>
                    <span class={`ml-1 text-xs tabular-nums ${on ? label : "text-slate-400"}`}>
                      {columns[key].all.length}
                    </span>
                  </button>
                );
              })}
            </div>
          </div>

          <div class="mt-3 grid gap-4 lg:mt-4 lg:grid-cols-3">
            {cfgT.map(({ key, meaning, bar, label }) => {
              const col = columns[key];
              const shown = expanded[key] ? col.all : col.capped;
              return (
                <div
                  class={`overflow-hidden rounded-xl border border-slate-200 bg-white ${
                    key === activeTier ? "" : "hidden lg:block"
                  }`}
                >
                <div class={`h-1 ${bar}`} />
                <div class="flex items-baseline justify-between px-3 pt-2.5">
                  <div class="flex items-baseline gap-1.5">
                    <span class={`text-base font-bold ${label}`}>{key}</span>
                    <span class="text-xs text-slate-400">{meaning}</span>
                  </div>
                  <span class="text-xs tabular-nums text-slate-400">{col.all.length} 个</span>
                </div>
                <div class="px-3 pb-1.5 pt-1 text-right text-[10px] tracking-wide text-slate-400">
                  {canScore ? "等效分 · 与你差距" : "录取位次 · 与你差距"}
                </div>
                <ul class="divide-y divide-slate-100 border-t border-slate-100">
                  {shown.map((e) => renderRow(e))}
                  {col.all.length === 0 && (
                    <li class="px-3 py-3 text-xs text-slate-300">
                      {activeFilters ? "这一档无符合筛选的" : "这一档暂无可填"}
                    </li>
                  )}
                </ul>
                {col.hasMore && !expanded[key] && (
                  <button
                    type="button"
                    onClick={() => setExpanded((p) => ({ ...p, [key]: true }))}
                    class="w-full border-t border-slate-100 px-3 py-2 text-xs text-slate-500 hover:bg-slate-50"
                  >
                    展开剩余 {col.all.length - col.capped.length} 个 ▾
                  </button>
                )}
                {col.far && (
                  <details class="border-t-2 border-dashed border-slate-300 bg-slate-100/70">
                    <summary class="flex cursor-pointer list-none flex-wrap items-center gap-x-1.5 gap-y-0.5 px-3 py-2.5 text-xs hover:bg-slate-200/60">
                      <span class="text-slate-400">▸</span>
                      <span class="rounded bg-slate-200 px-1.5 py-0.5 font-medium text-slate-600">{col.far.bucket}</span>
                      <span class="text-slate-500">{col.far.entries.length} 个 · 仅供参考</span>
                      <span class="w-full text-slate-400 sm:ml-auto sm:w-auto">{FAR_NOTE[col.far.bucket]}</span>
                    </summary>
                    <ul class="divide-y divide-slate-200/70 border-t border-dashed border-slate-300">
                      {col.far.entries.map((e) => renderRow(e, true))}
                    </ul>
                  </details>
                )}
              </div>
              );
            })}
          </div>
        </>
      )}
      {!loading && V <= 0 && (
        <div class="mt-6 rounded-xl border border-dashed border-slate-300 bg-white px-4 py-10 text-center">
          <p class="text-sm text-slate-500">
            输入分数或位次，按等效位次给出 <span class="font-medium text-rose-600">冲</span> /{" "}
            <span class="font-medium text-amber-600">稳</span> /{" "}
            <span class="font-medium text-emerald-600">保</span> 的可填报{unit}。
          </p>
        </div>
      )}
    </div>
  );
}

const LEVEL_RANK = ["985", "211", "双一流"];
function topLevel(lv?: string[]): string | undefined {
  if (!lv) return undefined;
  return LEVEL_RANK.find((t) => lv.includes(t));
}
function levelCls(lv: string): string {
  return lv === "985"
    ? "bg-indigo-50 text-indigo-600"
    : lv === "211"
      ? "bg-sky-50 text-sky-600"
      : "bg-violet-50 text-violet-700";
}

function distinctSorted(xs: (string | undefined)[]): string[] {
  const set = new Set<string>();
  for (const x of xs) if (x) set.add(x);
  return [...set].sort((a, b) => a.localeCompare(b, "zh"));
}

function ChipRow({
  label,
  options,
  selected,
  onToggle,
  labelOf,
}: {
  label: string;
  options: string[];
  selected: string[];
  onToggle: (v: string) => void;
  labelOf?: (v: string) => string;
}) {
  return (
    <div class="flex flex-col gap-1.5">
      <span class="text-xs font-medium text-slate-400">{label}</span>
      <div class="flex flex-wrap gap-1.5">
        {options.map((o) => {
          const on = selected.includes(o);
          return (
            <button
              type="button"
              key={o}
              onClick={() => onToggle(o)}
              class={`rounded-full px-2.5 py-1 text-xs font-medium transition ${
                on ? "bg-slate-800 text-white" : "border border-slate-300 text-slate-600 hover:border-slate-400"
              }`}
            >
              {labelOf ? labelOf(o) : o}
            </button>
          );
        })}
      </div>
    </div>
  );
}

function KeywordField({
  label,
  value,
  onInput,
  placeholder,
}: {
  label: string;
  value: string;
  onInput: (v: string) => void;
  placeholder: string;
}) {
  return (
    <div class="flex flex-col gap-1.5">
      <span class="text-xs font-medium text-slate-400">{label}</span>
      <input
        type="text"
        value={value}
        onInput={(e) => onInput((e.target as HTMLInputElement).value)}
        placeholder={placeholder}
        class="w-72 max-w-full rounded-lg border border-slate-300 px-3 py-1.5 text-sm focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
      />
    </div>
  );
}

function NumField({ label, value, onChange }: { label: string; value: number; onChange: (n: number) => void }) {
  return (
    <div class="flex flex-col gap-1.5">
      <span class="text-xs font-medium text-slate-400">{label}</span>
      <input
        type="number"
        inputMode="numeric"
        min={0}
        value={value > 0 ? String(value) : ""}
        onInput={(e) => {
          const n = parseInt((e.target as HTMLInputElement).value, 10);
          onChange(Number.isNaN(n) || n < 0 ? 0 : n);
        }}
        placeholder="不限"
        class="w-24 rounded-lg border border-slate-300 px-3 py-1.5 text-sm focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
      />
    </div>
  );
}
