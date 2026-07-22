import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { ReportScopeSelect } from "../ReportScopeSelect";
import {
  SCOPE_ALL,
  SCOPE_CONSOLIDATED,
  scopeToParams,
  scopeEntityId,
} from "../reportScope";
import { endpoints } from "@/lib/api";

vi.mock("@/lib/api", () => ({ endpoints: { getEntities: vi.fn() } }));

const wrapper = ({ children }) => (
  <QueryClientProvider
    client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
  >
    {children}
  </QueryClientProvider>
);

describe("reportScope helpers", () => {
  it("maps a scope to the right query params", () => {
    expect(scopeToParams(SCOPE_ALL)).toEqual({});
    expect(scopeToParams(SCOPE_CONSOLIDATED)).toEqual({ consolidated: true });
    expect(scopeToParams("entity-123")).toEqual({ entity_id: "entity-123" });
  });

  it("returns an entity id only for a single-entity scope", () => {
    expect(scopeEntityId(SCOPE_ALL)).toBe("");
    expect(scopeEntityId(SCOPE_CONSOLIDATED)).toBe("");
    expect(scopeEntityId("entity-123")).toBe("entity-123");
  });
});

describe("ReportScopeSelect", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders nothing for a single-entity tenant", async () => {
    endpoints.getEntities.mockResolvedValue({
      data: { data: [{ id: "1", name: "ACME", is_primary: true }] },
    });
    const { container } = render(<ReportScopeSelect value={SCOPE_ALL} onChange={() => {}} />, { wrapper });
    await waitFor(() => expect(endpoints.getEntities).toHaveBeenCalled());
    expect(container.querySelector('[role="combobox"]')).toBeNull();
  });

  it("offers All / Consolidated when multiple entities exist", async () => {
    endpoints.getEntities.mockResolvedValue({
      data: {
        data: [
          { id: "1", name: "ACME Inc", is_primary: true },
          { id: "2", name: "ACME UK", is_primary: false },
        ],
      },
    });
    render(<ReportScopeSelect value={SCOPE_ALL} onChange={() => {}} />, { wrapper });
    // The trigger shows the current "All entities" scope.
    await screen.findByText("All entities");
    expect(screen.getByText("Scope")).toBeInTheDocument();
  });
});
