import { useState } from "preact/hooks";

/**
 * 最小 Preact island，仅用于验证客户端交互通路（client:* 指令）。
 * 后续切片的位次定位 island 会替换它。
 */
export default function Ping() {
  const [n, setN] = useState(0);
  return (
    <button
      type="button"
      onClick={() => setN((c) => c + 1)}
      class="rounded-md bg-slate-900 px-3 py-1.5 text-sm text-white hover:bg-slate-700"
    >
      island 已挂载 · 点击 {n}
    </button>
  );
}
