// Deterministic ranking within one object result group. The backend already
// orders by ILIKE relevance; this re-orders the small returned set (≤ limit) so
// the most likely match is first, then STABLE-preserves the backend order within
// a tier. Kept intentionally simple — not a scoring engine.
//
// Tiers (best first):
//   0  exact id / code match
//   1  exact name match
//   2  name prefix match
//   3  name substring match
//   4  secondary-field match (code / email / etc.)
//   5  backend matched something we don't model (keep, after the above)
//
// `accessors` provides optional field readers: { id, code, name, secondary }.
export function rankResults(items, query, accessors = {}) {
  const q = String(query || "").trim().toLowerCase();
  if (!q) return items;
  const lc = (v) => String(v ?? "").toLowerCase();
  const read = (fn, item) => (fn ? lc(fn(item)) : "");

  const tierOf = (item) => {
    const id = read(accessors.id, item);
    const code = read(accessors.code, item);
    const name = read(accessors.name, item);
    const secondary = read(accessors.secondary, item);
    if ((id && id === q) || (code && code === q)) return 0;
    if (name && name === q) return 1;
    if (name && name.startsWith(q)) return 2;
    if (name && name.includes(q)) return 3;
    if ((code && code.includes(q)) || (secondary && secondary.includes(q))) return 4;
    return 5;
  };

  return items
    .map((item, i) => ({ item, i, t: tierOf(item) }))
    .sort((a, b) => a.t - b.t || a.i - b.i)
    .map((x) => x.item);
}
