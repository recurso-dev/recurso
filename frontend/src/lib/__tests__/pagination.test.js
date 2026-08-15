import { describe, it, expect, vi } from "vitest";
import {
  LIST_PAGE_SIZE,
  pageOffset,
  pageSlice,
  fetchAllPages,
} from "../pagination";

describe("pageOffset", () => {
  it("maps the 1-based page to a 0-based offset", () => {
    expect(pageOffset(1, 25)).toBe(0);
    expect(pageOffset(2, 25)).toBe(25);
    expect(pageOffset(4, 25)).toBe(75);
  });
  it("clamps a page below 1 to offset 0", () => {
    expect(pageOffset(0, 25)).toBe(0);
    expect(pageOffset(-3, 25)).toBe(0);
  });
  it("defaults to LIST_PAGE_SIZE", () => {
    expect(pageOffset(2)).toBe(LIST_PAGE_SIZE);
  });
});

describe("pageSlice", () => {
  const rows = Array.from({ length: 55 }, (_, i) => i); // 0..54

  it("returns the first page", () => {
    expect(pageSlice(rows, 1, 25)).toEqual(rows.slice(0, 25));
  });
  it("returns a middle page", () => {
    expect(pageSlice(rows, 2, 25)).toEqual(rows.slice(25, 50));
  });
  it("returns a short last page", () => {
    expect(pageSlice(rows, 3, 25)).toEqual([50, 51, 52, 53, 54]);
  });
  it("returns empty past the end", () => {
    expect(pageSlice(rows, 4, 25)).toEqual([]);
  });
  it("returns empty for an empty list", () => {
    expect(pageSlice([], 1, 25)).toEqual([]);
  });
});

describe("fetchAllPages", () => {
  it("stops at the first short page (complete set)", async () => {
    const fetchWindow = vi.fn(async (offset) => (offset === 0 ? [1, 2, 3] : []));
    const { rows, truncated } = await fetchAllPages(fetchWindow, { pageSize: 250 });
    expect(rows).toEqual([1, 2, 3]);
    expect(truncated).toBe(false);
    expect(fetchWindow).toHaveBeenCalledTimes(1);
  });

  it("pages through until a short page, concatenating windows", async () => {
    // 3 rows per window, two full windows then a short one.
    const pages = [
      [1, 2, 3],
      [4, 5, 6],
      [7],
    ];
    const fetchWindow = vi.fn(async (offset, limit) => pages[offset / limit] ?? []);
    const { rows, truncated } = await fetchAllPages(fetchWindow, { pageSize: 3 });
    expect(rows).toEqual([1, 2, 3, 4, 5, 6, 7]);
    expect(truncated).toBe(false);
    expect(fetchWindow).toHaveBeenCalledTimes(3);
  });

  it("reports truncated when the safety cap is hit with full pages", async () => {
    const fetchWindow = vi.fn(async () => [1, 2]); // always full
    const { rows, truncated } = await fetchAllPages(fetchWindow, {
      pageSize: 2,
      maxPages: 3,
    });
    expect(rows).toHaveLength(6); // 3 pages × 2
    expect(truncated).toBe(true);
    expect(fetchWindow).toHaveBeenCalledTimes(3);
  });

  it("tolerates a null/undefined window result", async () => {
    const fetchWindow = vi.fn(async () => null);
    const { rows, truncated } = await fetchAllPages(fetchWindow, { pageSize: 5 });
    expect(rows).toEqual([]);
    expect(truncated).toBe(false);
  });
});
