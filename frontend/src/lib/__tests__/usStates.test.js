import { describe, it, expect } from "vitest";
import { US_STATES, US_STATE_NAME } from "../usStates";

describe("US_STATES", () => {
  it("covers all 50 states + DC", () => {
    expect(US_STATES).toHaveLength(51);
  });

  it("has unique, valid 2-letter uppercase codes", () => {
    const codes = US_STATES.map((s) => s.code);
    expect(new Set(codes).size).toBe(51);
    for (const c of codes) expect(c).toMatch(/^[A-Z]{2}$/);
  });

  it("includes DC and representative states", () => {
    const codes = new Set(US_STATES.map((s) => s.code));
    for (const c of ["DC", "CA", "NY", "TX", "WY", "HI", "AK"]) {
      expect(codes.has(c)).toBe(true);
    }
  });

  it("flags exactly the four no-statewide-sales-tax states", () => {
    const noTax = US_STATES.filter((s) => s.noSalesTax).map((s) => s.code).sort();
    expect(noTax).toEqual(["DE", "MT", "NH", "OR"]);
  });

  it("resolves codes to names", () => {
    expect(US_STATE_NAME.CA).toBe("California");
    expect(US_STATE_NAME.DC).toBe("District of Columbia");
    expect(US_STATE_NAME.OR).toBe("Oregon");
  });
});
