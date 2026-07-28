import { describe, it, expect } from "vitest";
import { COUNTRIES, COUNTRY_NAME } from "../countries";

describe("countries", () => {
  it("exposes a non-empty list of {code,name} entries", () => {
    expect(Array.isArray(COUNTRIES)).toBe(true);
    expect(COUNTRIES.length).toBeGreaterThan(0);
    for (const c of COUNTRIES) {
      expect(typeof c.code).toBe("string");
      expect(c.code).toMatch(/^[A-Z]{2}$/); // ISO-3166 alpha-2
      expect(typeof c.name).toBe("string");
      expect(c.name.length).toBeGreaterThan(0);
    }
  });

  it("has unique country codes", () => {
    const codes = COUNTRIES.map((c) => c.code);
    expect(new Set(codes).size).toBe(codes.length);
  });

  it("COUNTRY_NAME maps every code to its name", () => {
    for (const c of COUNTRIES) {
      expect(COUNTRY_NAME[c.code]).toBe(c.name);
    }
    // A couple of well-known anchors.
    expect(COUNTRY_NAME.US).toBeTruthy();
    expect(COUNTRY_NAME.IN).toBeTruthy();
  });
});
