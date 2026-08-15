import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router";

import { compareValues, sortRows, parseSort, useTableSort } from "../tableSort";

describe("compareValues", () => {
  it("orders numbers numerically, not lexically", () => {
    // 9 vs 100: a string compare would put "100" before "9".
    expect(compareValues(9, 100, 1)).toBeLessThan(0);
    expect(compareValues(100, 9, 1)).toBeGreaterThan(0);
  });

  it("respects direction for numbers", () => {
    expect(compareValues(1, 2, -1)).toBeGreaterThan(0); // desc: 1 after 2
  });

  it("compares strings case-insensitively via localeCompare", () => {
    expect(compareValues("apple", "banana", 1)).toBeLessThan(0);
  });

  it("always sorts nulls last, regardless of direction", () => {
    expect(compareValues(null, 5, 1)).toBeGreaterThan(0);
    expect(compareValues(null, 5, -1)).toBeGreaterThan(0);
    expect(compareValues(5, null, -1)).toBeLessThan(0);
    expect(compareValues(null, null, 1)).toBe(0);
  });
});

const COLUMNS = [
  { key: "amount", sortable: true, sortValue: (r) => r.amount },
  { key: "name", sortable: true, sortValue: (r) => r.name },
  { key: "status", sortable: true }, // no sortValue → row[key]
  { key: "note", sortable: false },
];

const ROWS = [
  { id: "a", amount: 900, name: "Charlie", status: "open", note: "z" },
  { id: "b", amount: 100, name: "alice", status: "paid", note: "a" },
  { id: "c", amount: 5000, name: "Bob", status: "open", note: "m" },
];

describe("sortRows", () => {
  it("returns the original array (untouched) when there is no active sort", () => {
    const out = sortRows(ROWS, null, COLUMNS);
    expect(out).toBe(ROWS);
  });

  it("returns the original array for an unknown or non-sortable column", () => {
    expect(sortRows(ROWS, { key: "missing", dir: "asc" }, COLUMNS)).toBe(ROWS);
    expect(sortRows(ROWS, { key: "note", dir: "asc" }, COLUMNS)).toBe(ROWS);
  });

  it("sorts by a numeric column ascending and descending", () => {
    const asc = sortRows(ROWS, { key: "amount", dir: "asc" }, COLUMNS).map((r) => r.id);
    expect(asc).toEqual(["b", "a", "c"]);
    const desc = sortRows(ROWS, { key: "amount", dir: "desc" }, COLUMNS).map((r) => r.id);
    expect(desc).toEqual(["c", "a", "b"]);
  });

  it("sorts by a string column case-insensitively", () => {
    const asc = sortRows(ROWS, { key: "name", dir: "asc" }, COLUMNS).map((r) => r.id);
    expect(asc).toEqual(["b", "c", "a"]); // alice, Bob, Charlie
  });

  it("falls back to row[key] when the column has no sortValue", () => {
    const asc = sortRows(ROWS, { key: "status", dir: "asc" }, COLUMNS).map((r) => r.status);
    expect(asc).toEqual(["open", "open", "paid"]);
  });

  it("does not mutate the input array", () => {
    const copy = [...ROWS];
    sortRows(ROWS, { key: "amount", dir: "desc" }, COLUMNS);
    expect(ROWS).toEqual(copy);
  });

  it("places rows with a null accessor value last", () => {
    const rows = [{ id: "x", amount: 5 }, { id: "y", amount: null }, { id: "z", amount: 1 }];
    const asc = sortRows(rows, { key: "amount", dir: "asc" }, COLUMNS).map((r) => r.id);
    expect(asc).toEqual(["z", "x", "y"]);
    const desc = sortRows(rows, { key: "amount", dir: "desc" }, COLUMNS).map((r) => r.id);
    expect(desc).toEqual(["x", "z", "y"]); // null still last
  });
});

describe("parseSort", () => {
  it("parses a valid key:dir", () => {
    expect(parseSort("amount:desc")).toEqual({ key: "amount", dir: "desc" });
  });
  it("rejects empty, malformed, or invalid direction", () => {
    expect(parseSort("")).toBeNull();
    expect(parseSort(null)).toBeNull();
    expect(parseSort("amount")).toBeNull();
    expect(parseSort("amount:sideways")).toBeNull();
    expect(parseSort(":asc")).toBeNull();
  });
});

function Probe() {
  const { sort, sortKey, onSortChange } = useTableSort();
  const loc = useLocation();
  return (
    <div>
      <span data-testid="sort">{JSON.stringify(sort)}</span>
      <span data-testid="sortKey">{sortKey}</span>
      <span data-testid="search">{loc.search}</span>
      <button onClick={() => onSortChange({ key: "amount", dir: "desc" })}>setAmountDesc</button>
      <button onClick={() => onSortChange({ key: "amount", dir: "asc" })}>setAmountAsc</button>
      <button onClick={() => onSortChange(null)}>clear</button>
    </div>
  );
}

const renderAt = (route) =>
  render(
    <MemoryRouter initialEntries={[route]}>
      <Probe />
    </MemoryRouter>,
  );

describe("useTableSort", () => {
  it("is unsorted by default (no sort param, clean URL)", () => {
    renderAt("/invoices");
    expect(screen.getByTestId("sort")).toHaveTextContent("null");
    expect(screen.getByTestId("search")).toHaveTextContent("");
  });

  it("reads an existing sort param from the URL", () => {
    renderAt("/invoices?sort=amount:desc");
    expect(screen.getByTestId("sort")).toHaveTextContent('{"key":"amount","dir":"desc"}');
    expect(screen.getByTestId("sortKey")).toHaveTextContent("amount:desc");
  });

  it("ignores a malformed sort param (unsorted)", () => {
    renderAt("/invoices?sort=bogus");
    expect(screen.getByTestId("sort")).toHaveTextContent("null");
  });

  it("writes the sort to the URL as key:dir", () => {
    renderAt("/invoices");
    fireEvent.click(screen.getByText("setAmountDesc"));
    expect(screen.getByTestId("search")).toHaveTextContent("sort=amount%3Adesc");
    expect(screen.getByTestId("sort")).toHaveTextContent('{"key":"amount","dir":"desc"}');
  });

  it("clears the sort param when set to null", () => {
    renderAt("/invoices?sort=amount:asc");
    fireEvent.click(screen.getByText("clear"));
    expect(screen.getByTestId("search")).not.toHaveTextContent("sort=");
    expect(screen.getByTestId("sort")).toHaveTextContent("null");
  });

  it("preserves other list params (search/filter/page) when changing sort", () => {
    renderAt("/invoices?q=globex&status=open&page=2");
    fireEvent.click(screen.getByText("setAmountAsc"));
    const search = screen.getByTestId("search").textContent;
    expect(search).toContain("q=globex");
    expect(search).toContain("status=open");
    expect(search).toContain("sort=amount%3Aasc");
  });
});
