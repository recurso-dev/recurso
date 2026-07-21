import { describe, it, expect } from "vitest";
import { currencyDecimals, toMinorUnits, fromMinorUnits, formatCurrency } from "../utils";

describe("currency-aware money helpers", () => {
    it("derives decimals from the currency", () => {
        expect(currencyDecimals("USD")).toBe(2);
        expect(currencyDecimals("EUR")).toBe(2);
        expect(currencyDecimals("INR")).toBe(2);
        expect(currencyDecimals("JPY")).toBe(0);
        expect(currencyDecimals("KWD")).toBe(3);
        expect(currencyDecimals("bogus")).toBe(2); // safe fallback
    });

    it("toMinorUnits respects the currency exponent", () => {
        expect(toMinorUnits(42, "USD")).toBe(4200);
        expect(toMinorUnits("42", "USD")).toBe(4200);
        expect(toMinorUnits(1000, "JPY")).toBe(1000); // 0-decimal — not *100
        expect(toMinorUnits(4.2, "KWD")).toBe(4200); // 3-decimal
        expect(toMinorUnits("", "USD")).toBe(0);
    });

    it("fromMinorUnits inverts toMinorUnits", () => {
        for (const [cur, major] of [["USD", 42], ["JPY", 1000], ["KWD", 4.2]]) {
            expect(fromMinorUnits(toMinorUnits(major, cur), cur)).toBeCloseTo(major, 10);
        }
    });

    it("formatCurrency shows the currency's native decimals", () => {
        expect(formatCurrency(4200, "USD")).toBe("$42.00");
        expect(formatCurrency(1000, "JPY")).toBe("¥1,000"); // no decimals
        // USD default is unchanged (no regression for the common case)
        expect(formatCurrency(150000, "USD")).toBe("$1,500.00");
    });
});
