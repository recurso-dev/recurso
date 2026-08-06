import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import SettingsLayout from "../SettingsLayout";
import { endpoints } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  endpoints: { getEntities: vi.fn() },
}));

const primary = (country) => ({
  data: {
    data: [{ id: "ent-1", is_primary: true, country_code: country }],
  },
});

const renderAt = (path) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/settings" element={<SettingsLayout />}>
            <Route index element={<div>GENERAL CONTENT</div>} />
            <Route path="tax-nexus" element={<div>NEXUS CONTENT</div>} />
          </Route>
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );

describe("SettingsLayout — persistent settings nav", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getEntities.mockResolvedValue(primary("US"));
  });

  it("shows the section nav alongside the routed content (Outlet)", async () => {
    renderAt("/settings/tax-nexus");
    // The nav is persistent...
    expect(screen.getByRole("link", { name: /general/i })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /billing & plan/i })).toBeInTheDocument();
    // ...and the routed sub-page content renders beside it.
    expect(screen.getByText("NEXUS CONTENT")).toBeInTheDocument();
  });

  it("marks the active section", async () => {
    renderAt("/settings/tax-nexus");
    const active = screen.getByRole("link", { name: /US sales-tax nexus/i });
    expect(active).toHaveAttribute("aria-current", "page");
  });

  it("badges the region-relevant tax setups for a US business", async () => {
    renderAt("/settings");
    await waitFor(() => expect(screen.getAllByText("For your region")).toHaveLength(2));
    const badged = screen.getAllByText("For your region").map((b) => b.closest("a").textContent);
    expect(badged.some((t) => t.includes("US sales-tax nexus"))).toBe(true);
    expect(badged.some((t) => t.includes("US tax identity"))).toBe(true);
  });

  it("badges GST + IRP for an India business", async () => {
    endpoints.getEntities.mockResolvedValue(primary("IN"));
    renderAt("/settings");
    await waitFor(() => expect(screen.getAllByText("For your region")).toHaveLength(2));
    const badged = screen.getAllByText("For your region").map((b) => b.closest("a").textContent);
    expect(badged.some((t) => t.includes("GST configuration"))).toBe(true);
    expect(badged.some((t) => t.includes("E-invoicing (IRP)"))).toBe(true);
  });
});
