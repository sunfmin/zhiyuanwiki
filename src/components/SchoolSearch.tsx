import { useMemo, useState } from "preact/hooks";

type S = { code: string; name: string; leafCount: number };
const CAP = 300;

/** 按院校中文名（或代码）站内搜索，链接到 /yuanxiao/{code}/。 */
export default function SchoolSearch({ schools }: { schools: S[] }) {
  const [q, setQ] = useState("");

  const list = useMemo(() => {
    const t = q.trim();
    const base = t ? schools.filter((s) => s.name.includes(t) || s.code.includes(t)) : schools;
    return base.slice(0, CAP);
  }, [q, schools]);

  const matched = q.trim() ? schools.filter((s) => s.name.includes(q.trim()) || s.code.includes(q.trim())).length : schools.length;

  return (
    <div>
      <input
        type="search"
        value={q}
        onInput={(e) => setQ((e.target as HTMLInputElement).value)}
        placeholder="按院校名或代码搜索，如 哈尔滨工业大学"
        class="w-full max-w-md rounded-md border border-slate-300 px-3 py-2"
      />
      <p class="mt-2 text-sm text-slate-500">
        {q.trim()
          ? `匹配 ${matched} 所${matched > CAP ? `（显示前 ${CAP}）` : ""}`
          : `共 ${schools.length} 所院校`}
      </p>

      <ul class="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {list.map((s) => (
          <li>
            <a
              href={`/yuanxiao/${s.code}/`}
              class="flex items-baseline justify-between rounded-lg border border-slate-200 bg-white px-3 py-2 hover:border-slate-400"
            >
              <span class="font-medium">{s.name}</span>
              <span class="ml-2 shrink-0 text-xs text-slate-400">
                {s.code} · {s.leafCount} 专业
              </span>
            </a>
          </li>
        ))}
        {list.length === 0 && <li class="text-sm text-slate-400">没有匹配的院校</li>}
      </ul>
    </div>
  );
}
