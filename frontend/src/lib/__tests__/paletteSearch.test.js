import { describe, it, expect } from "vitest";
import { rankResults } from "../paletteSearch";

const acc = { id: (x) => x.id, code: (x) => x.code, name: (x) => x.name, secondary: (x) => x.email };

describe("rankResults", () => {
  it("returns items unchanged for an empty query", () => {
    const items = [{ name: "b" }, { name: "a" }];
    expect(rankResults(items, "", acc)).toBe(items);
  });

  it("ranks an exact name match above a prefix and a substring match", () => {
    const items = [
      { id: "3", name: "The Acme Group" }, // substring (does NOT start with the query)
      { id: "2", name: "Acme" }, // exact
      { id: "1", name: "Acme Corp" }, // prefix
    ];
    const out = rankResults(items, "acme", acc).map((x) => x.name);
    expect(out).toEqual(["Acme", "Acme Corp", "The Acme Group"]);
  });

  it("ranks an exact id/code match first", () => {
    const items = [
      { id: "x", code: "PRO", name: "Pro Plan" },
      { id: "PRO-USD", code: "PRO-USD", name: "Something else" },
    ];
    const out = rankResults(items, "pro-usd", acc).map((x) => x.id);
    expect(out[0]).toBe("PRO-USD");
  });

  it("ranks a secondary-field (email) match after name matches", () => {
    const items = [
      { id: "1", name: "Zeta", email: "acme@x.com" }, // secondary match only
      { id: "2", name: "Acme Ltd", email: "z@z.com" }, // prefix name match
    ];
    const out = rankResults(items, "acme", acc).map((x) => x.name);
    expect(out).toEqual(["Acme Ltd", "Zeta"]);
  });

  it("is stable — preserves backend order within a tier", () => {
    const items = [
      { id: "1", name: "Acme One" },
      { id: "2", name: "Acme Two" },
      { id: "3", name: "Acme Three" },
    ];
    // All are prefix matches (tier 2) → order preserved.
    const out = rankResults(items, "acme", acc).map((x) => x.id);
    expect(out).toEqual(["1", "2", "3"]);
  });
});
