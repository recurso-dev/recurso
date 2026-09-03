import { describe, it, expect } from "vitest";
import {
  cn,
  currencyDecimals,
  fromMinorUnits,
  toMinorUnits,
  formatCurrency,
  formatCurrencyHeadline,
  formatNumber,
  formatDate,
  shortId,
} from "../utils";

describe("cn", () => {
  it("merges class names and resolves Tailwind conflicts", () => {
    expect(cn("px-2", "py-1")).toBe("px-2 py-1");
    // twMerge keeps the last conflicting utility.
    expect(cn("px-2", "px-4")).toBe("px-4");
  });

  it("drops falsy values", () => {
    // eslint-disable-next-line no-constant-binary-expression -- the falsy `&&` is the input under test
    expect(cn("a", false && "b", null, undefined, "c")).toBe("a c");
  });
});

describe("currencyDecimals", () => {
  it("returns the real exponent per currency", () => {
    expect(currencyDecimals("USD")).toBe(2);
    expect(currencyDecimals("EUR")).toBe(2);
    expect(currencyDecimals("JPY")).toBe(0);
    expect(currencyDecimals("KRW")).toBe(0);
    expect(currencyDecimals("KWD")).toBe(3);
    expect(currencyDecimals("BHD")).toBe(3);
  });

  it("defaults to 2 for unknown/invalid/empty codes", () => {
    expect(currencyDecimals("")).toBe(2);
    expect(currencyDecimals(undefined)).toBe(2);
    expect(currencyDecimals("NOTACURRENCY")).toBe(2);
  });
});

describe("fromMinorUnits", () => {
  it("converts using the currency exponent", () => {
    expect(fromMinorUnits(4200, "USD")).toBe(42);
    expect(fromMinorUnits(4200, "JPY")).toBe(4200);
    expect(fromMinorUnits(4200, "KWD")).toBe(4.2);
  });

  it("handles zero, negatives, and non-numeric input", () => {
    expect(fromMinorUnits(0, "USD")).toBe(0);
    expect(fromMinorUnits(-4200, "USD")).toBe(-42);
    expect(fromMinorUnits(null, "USD")).toBe(0);
    expect(fromMinorUnits(undefined, "USD")).toBe(0);
    expect(fromMinorUnits("bad", "USD")).toBe(0);
    expect(fromMinorUnits("4200", "USD")).toBe(42); // numeric string
  });

  it("defaults currency to USD", () => {
    expect(fromMinorUnits(100)).toBe(1);
  });
});

describe("toMinorUnits", () => {
  it("converts major-unit input using the currency exponent", () => {
    expect(toMinorUnits(42, "USD")).toBe(4200);
    expect(toMinorUnits(4200, "JPY")).toBe(4200);
    expect(toMinorUnits(4.2, "KWD")).toBe(4200);
  });

  it("rounds to the nearest minor unit", () => {
    expect(toMinorUnits(42.005, "USD")).toBe(4201); // 4200.5 rounds up
    expect(toMinorUnits(42.004, "USD")).toBe(4200);
  });

  it("handles empty/invalid input as 0", () => {
    expect(toMinorUnits("", "USD")).toBe(0);
    expect(toMinorUnits(null, "USD")).toBe(0);
    expect(toMinorUnits("abc", "USD")).toBe(0);
  });

  it("round-trips with fromMinorUnits across currencies", () => {
    for (const [minor, cur] of [
      [4200, "USD"],
      [4200, "JPY"],
      [4200, "KWD"],
      [1, "USD"],
      [999999, "BHD"],
    ]) {
      expect(toMinorUnits(fromMinorUnits(minor, cur), cur)).toBe(minor);
    }
  });
});

describe("formatCurrency", () => {
  it("formats minor units with the currency's own decimals", () => {
    expect(formatCurrency(4200, "USD")).toBe("$42.00");
    expect(formatCurrency(25000, "USD")).toBe("$250.00");
    // JPY has no minor unit — 4200 yen shows as ¥4,200, not ¥42.
    expect(formatCurrency(4200, "JPY")).toBe("¥4,200");
  });

  it("formats negatives and zero", () => {
    expect(formatCurrency(-8200, "USD")).toBe("-$82.00");
    expect(formatCurrency(0, "USD")).toBe("$0.00");
  });

  it("defaults currency to USD and tolerates nullish amount", () => {
    expect(formatCurrency(100)).toBe("$1.00");
    expect(formatCurrency(null, "USD")).toBe("$0.00");
  });
});

describe("formatCurrencyHeadline", () => {
  it("drops the .00 tail on whole amounts", () => {
    expect(formatCurrencyHeadline(1867500, "USD")).toBe("$18,675");
    expect(formatCurrencyHeadline(100000, "USD")).toBe("$1,000");
  });

  it("keeps real cents on non-whole amounts", () => {
    expect(formatCurrencyHeadline(-82, "USD")).toBe("-$0.82");
    expect(formatCurrencyHeadline(150050, "USD")).toBe("$1,500.50");
  });

  it("handles zero-decimal currencies as whole", () => {
    expect(formatCurrencyHeadline(4200, "JPY")).toBe("¥4,200");
  });
});

describe("formatNumber", () => {
  it("groups thousands by default", () => {
    expect(formatNumber(1234567)).toBe("1,234,567");
  });

  it("respects Intl options", () => {
    expect(formatNumber(0.1234, { style: "percent" })).toBe("12%");
  });

  it("coerces invalid input to 0", () => {
    expect(formatNumber("bad")).toBe("0");
    expect(formatNumber(null)).toBe("0");
  });
});

describe("formatDate", () => {
  it("formats an ISO string", () => {
    expect(formatDate("2026-01-15T00:00:00Z")).toMatch(/2026/);
  });

  it("accepts a Date instance", () => {
    expect(formatDate(new Date("2026-03-01T12:00:00Z"))).toMatch(/2026/);
  });

  it("returns an em-dash for empty/invalid input", () => {
    expect(formatDate("")).toBe("—");
    expect(formatDate(null)).toBe("—");
    expect(formatDate("not-a-date")).toBe("—");
  });
});

describe("shortId", () => {
  it("truncates a uuid to 8 chars with an ellipsis", () => {
    expect(shortId("abcd1234-5678-90ab-cdef-1234567890ab")).toBe("abcd1234…");
  });

  it("renders an em-dash for a missing id", () => {
    expect(shortId(null)).toBe("—");
    expect(shortId(undefined)).toBe("—");
    expect(shortId("")).toBe("—");
  });
});
