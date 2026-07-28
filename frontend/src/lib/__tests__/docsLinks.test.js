import { describe, it, expect } from "vitest";
import { docsUrlFor, DOCS_BASE, DOCS_HOME } from "../docsLinks";

describe("docsUrlFor", () => {
  it("maps the home route to the overview guide", () => {
    expect(docsUrlFor("/")).toBe(`${DOCS_BASE}/dashboard/overview`);
  });

  it("maps a simple first-segment route", () => {
    expect(docsUrlFor("/customers")).toBe(`${DOCS_BASE}/dashboard/customers`);
    expect(docsUrlFor("/customers/cus_123")).toBe(`${DOCS_BASE}/dashboard/customers`);
  });

  it("prefers a full-path mapping for nested finance routes", () => {
    expect(docsUrlFor("/finance/gst-returns")).toBe(`${DOCS_BASE}/compliance/gst-returns`);
    expect(docsUrlFor("/finance/entities")).toBe(`${DOCS_BASE}/dashboard/entities`);
  });

  it("falls back to the docs home for unmapped routes", () => {
    expect(docsUrlFor("/some/unknown/route")).toBe(DOCS_HOME);
  });
});
