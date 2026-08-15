// The one list page size, so every paginated list is consistent
// (QUALITY_BAR "Tables": "Page size must be consistent"). Server-side lists
// request LIST_PAGE_SIZE rows per page; the DataTable pagination contract turns
// the page into limit/offset (or page/per_page) at the call site.
export const LIST_PAGE_SIZE = 25;

// (page, size) → the zero-based offset for a limit/offset endpoint. `page` is
// 1-based everywhere in the UI (URL, DataTable), so page 1 → offset 0.
export const pageOffset = (page, size = LIST_PAGE_SIZE) => (Math.max(1, page) - 1) * size;

// Page-through a list endpoint that CANNOT filter/aggregate server-side, so the
// page must hold the complete set (client search, status chips, or StatCards
// that sum every row). This eliminates the silent truncation at the backend's
// default limit WITHOUT faking a total the API doesn't return.
//
// It is bounded: it stops at a short page or after `maxPages`, and reports
// `truncated` so the caller can say "showing the first N — refine or use the
// API" rather than pretending the set is complete. It does NOT solve backend
// scalability — a genuinely large list needs server-side pagination + a total
// count on the endpoint (tracked as a BACKEND GAP in the dashboard audit).
//
// `fetchWindow(offset, limit)` resolves to the row array for that window.
export async function fetchAllPages(
  fetchWindow,
  { pageSize = 250, maxPages = 20 } = {},
) {
  const rows = [];
  for (let i = 0; i < maxPages; i++) {
    const batch = (await fetchWindow(i * pageSize, pageSize)) || [];
    rows.push(...batch);
    if (batch.length < pageSize) return { rows, truncated: false };
  }
  return { rows, truncated: true };
}

// Slice a fully-loaded array to the current 1-based page. Used by the
// page-through lists to add pagination UI over their client-filtered rows.
export const pageSlice = (rows, page, size = LIST_PAGE_SIZE) => {
  const start = pageOffset(page, size);
  return rows.slice(start, start + size);
};
