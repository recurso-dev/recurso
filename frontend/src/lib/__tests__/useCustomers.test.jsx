import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { useCustomers, usePlans, useSubscriptions } from "../useCustomers";
import { endpoints } from "../api";

vi.mock("../api", () => ({
  endpoints: {
    getCustomers: vi.fn(),
    getPlans: vi.fn(),
    getSubscriptions: vi.fn(),
  },
}));

const wrapper = ({ children }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
    {children}
  </QueryClientProvider>
);

describe("useCustomers", () => {
  beforeEach(() => vi.clearAllMocks());

  it("builds an id→name map and requests the full set (anti-truncation)", async () => {
    endpoints.getCustomers.mockResolvedValue({
      data: { data: [{ id: "c1", name: "Acme" }, { id: "c2", name: "Beta" }] },
    });
    const { result } = renderHook(() => useCustomers(), { wrapper });
    await waitFor(() => expect(result.current.customers.length).toBe(2));
    expect(result.current.names).toEqual({ c1: "Acme", c2: "Beta" });
    // Must ask for everything — the API default (limit=10) would truncate.
    expect(endpoints.getCustomers).toHaveBeenCalledWith({ limit: 1000 });
  });

  it("degrades to empty on failure (best-effort)", async () => {
    endpoints.getCustomers.mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useCustomers(), { wrapper });
    await waitFor(() => expect(endpoints.getCustomers).toHaveBeenCalled());
    expect(result.current.customers).toEqual([]);
    expect(result.current.names).toEqual({});
  });
});

describe("usePlans", () => {
  beforeEach(() => vi.clearAllMocks());

  it("returns plans + names and asks for the full set", async () => {
    endpoints.getPlans.mockResolvedValue({ data: { data: [{ id: "p1", name: "Pro" }] } });
    const { result } = renderHook(() => usePlans(), { wrapper });
    await waitFor(() => expect(result.current.plans.length).toBe(1));
    expect(result.current.names).toEqual({ p1: "Pro" });
    expect(endpoints.getPlans).toHaveBeenCalledWith({ limit: 1000 });
  });
});

describe("useSubscriptions", () => {
  beforeEach(() => vi.clearAllMocks());

  it("returns the full subscription set", async () => {
    endpoints.getSubscriptions.mockResolvedValue({
      data: { data: [{ id: "s1" }, { id: "s2" }] },
    });
    const { result } = renderHook(() => useSubscriptions(), { wrapper });
    await waitFor(() => expect(result.current.length).toBe(2));
    expect(endpoints.getSubscriptions).toHaveBeenCalledWith({ limit: 1000 });
  });
});
