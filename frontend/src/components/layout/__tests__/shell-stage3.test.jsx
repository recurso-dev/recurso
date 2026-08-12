import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect } from "vitest";

import NotFound from "@/pages/NotFound";
import { labelForPath, ALL_DESTINATIONS, NAV_GROUPS } from "@/lib/navigation";

describe("navigation canon (lib/navigation.js)", () => {
  it("names every destination once — no duplicates, no Products alias", () => {
    const paths = ALL_DESTINATIONS.map((d) => d.to);
    expect(new Set(paths).size).toBe(paths.length);
    expect(paths).not.toContain("/products");
  });

  it("labels Collections (the page the old TITLES map missed)", () => {
    expect(labelForPath("/collections")).toBe("Collections");
  });

  it("resolves nested paths by longest prefix", () => {
    expect(labelForPath("/dunning/campaigns")).toBe("Dunning Campaigns");
    expect(labelForPath("/dunning")).toBe("Dunning");
    expect(labelForPath("/settings/gst")).toBe("GST Configuration");
  });

  it("marks /dunning for exact matching so two rows never highlight at once", () => {
    const dunning = NAV_GROUPS.flatMap((g) => g.items).find((i) => i.to === "/dunning");
    expect(dunning.end).toBe(true);
  });

  it("keeps Gifts and Referrals side by side and Audit Log in System", () => {
    const growth = NAV_GROUPS.find((g) => g.label === "Growth");
    expect(growth.items.map((i) => i.label)).toEqual(["Gifts", "Referrals"]);
    const system = NAV_GROUPS.find((g) => g.label === "System");
    expect(system.items.map((i) => i.label)).toContain("Audit Log");
    const books = NAV_GROUPS.find((g) => g.label === "Books");
    expect(books.items.map((i) => i.label)).not.toContain("Audit Log");
  });
});

describe("NotFound", () => {
  it("names the bad URL instead of silently redirecting", () => {
    render(
      <MemoryRouter initialEntries={["/customers/deadbeef"]}>
        <NotFound />
      </MemoryRouter>
    );
    expect(screen.getByRole("heading", { name: "Page not found" })).toBeInTheDocument();
    expect(screen.getByText("/customers/deadbeef")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Go to Home" })).toHaveAttribute("href", "/");
  });
});
