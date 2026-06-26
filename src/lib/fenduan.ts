// 一分一段：分数 ↔ 位次换算。纯函数，与 Go internal/hlj 逻辑镜像，客户端用。

export interface FenduanEntry {
  score: number;
  count: number;
  cumulative: number; // 累计人数 = 位次
}

export interface YiFenYiDuan {
  province: string;
  track: string;
  year: number;
  entries: FenduanEntry[]; // 按 score 升序
}

/**
 * 分数 → 位次：取"分数 ≥ score 的最小分数段"的累计人数。
 * 正确处理顶段（X以上）、缺失分（就近向上）、高于顶段（返回顶段累计）。
 */
export function scoreToRank(t: YiFenYiDuan, score: number): number | null {
  const e = t.entries;
  if (e.length === 0) return null;
  // 二分找第一个 score' >= score
  let lo = 0;
  let hi = e.length;
  while (lo < hi) {
    const mid = (lo + hi) >> 1;
    if (e[mid].score >= score) hi = mid;
    else lo = mid + 1;
  }
  if (lo === e.length) return e[e.length - 1].cumulative;
  return e[lo].cumulative;
}

/**
 * 位次 → 分数：取"累计人数 ≥ rank 的最高分数段"。
 */
export function rankToScore(t: YiFenYiDuan, rank: number): number | null {
  const e = t.entries;
  if (e.length === 0 || rank < 1) return null;
  for (let i = e.length - 1; i >= 0; i--) {
    if (e[i].cumulative >= rank) return e[i].score;
  }
  return e[0].score;
}
