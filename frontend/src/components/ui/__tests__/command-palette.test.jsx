import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";

import { CommandPalette } from "../command-palette";
import { endpoints } from "../../../lib/api";

vi.mock("../../../lib/api", () => ({
  endpoints: {
    getCustomers: vi.fn(),
    getPlans: vi.fn(),
    getSubscriptions: vi.fn(),
  },
}));

const list = (rows) => ({ data: { data: rows } });

function LocationProbe() {
  const l = useLocation();
  return <div data-testid="loc">{l.pathname + l.search}</div>;
}

function renderPalette({ onOpenChange = () => {}, seed } = {}) {
  // No gcTime:0 — seeded caches (customers/plans "all") must survive until the
  // palette's enabled:false readers observe them.
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  if (seed?.customers) client.setQueryData(["customers", "all"], seed.customers);
  if (seed?.plans) client.setQueryData(["plans", "all"], seed.plans);
  return render(
    <MemoryRouter initialEntries={["/start"]}>
      <QueryClientProvider client={client}>
        <CommandPalette open onOpenChange={onOpenChange} />
        <LocationProbe />
      </QueryClientProvider>
    </MemoryRouter>
  );
}

const input = () => screen.getByRole("combobox", { name: "Search Recurso" });

beforeEach(() => {
  vi.clearAllMocks();
  endpoints.getCustomers.mockResolvedValue(list([]));
  endpoints.getPlans.mockResolvedValue(list([]));
  endpoints.getSubscriptions.mockResolvedValue(list([]));
});

describe("CommandPalette — search", () => {
  it("does not query the backend for queries shorter than 2 characters", async () => {
    const user = userEvent.setup();
    renderPalette();
    await user.type(input(), "a");
    // Give the debounce time to settle.
    await new Promise((r) => setTimeout(r, 300));
    expect(endpoints.getCustomers).not.toHaveBeenCalled();
    expect(endpoints.getSubscriptions).not.toHaveBeenCalled();
  });

  it("searches Customers and renders identity + context, with a propagated AbortSignal", async () => {
    endpoints.getCustomers.mockResolvedValue(
      list([{ id: "c1", name: "Acme Corporation", email: "acme@x.com" }])
    );
    const user = userEvent.setup();
    renderPalette();
    await user.type(input(), "acme");
    await waitFor(() => expect(screen.getByText("Acme Corporation")).toBeInTheDocument());
    expect(screen.getByText("acme@x.com")).toBeInTheDocument();
    expect(screen.getByText("Customers")).toBeInTheDocument();
    // AbortSignal is propagated through the API abstraction.
    expect(endpoints.getCustomers).toHaveBeenCalledWith(
      expect.objectContaining({ q: "acme", limit: 6 }),
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    );
  });

  it("searches Plans and Subscriptions, resolving subscription names from the warm cache", async () => {
    endpoints.getPlans.mockResolvedValue(list([{ id: "p1", name: "Pro", code: "PRO-USD" }]));
    endpoints.getSubscriptions.mockResolvedValue(
      list([{ id: "s1", plan_id: "p1", customer_id: "c1", status: "active" }])
    );
    const user = userEvent.setup();
    renderPalette({
      seed: {
        customers: [{ id: "c1", name: "Acme Corporation" }],
        plans: [{ id: "p1", name: "Pro" }],
      },
    });
    await user.type(input(), "acme");
    // Wait on an OBJECT-specific token (the plan code) — "Plans" alone is also a
    // nav label, so it appears before the debounced object search resolves.
    await waitFor(() => expect(screen.getByText(/PRO-USD/)).toBeInTheDocument());
    // Subscription result carries its status from the search response…
    expect(screen.getByText(/active/i)).toBeInTheDocument();
    // …and its customer name is resolved from the warm cache (getCustomers
    // search returns [], so this name can only come from the ["customers","all"]
    // cache seed via the subscription's secondary line).
    expect(screen.getByText("Acme Corporation")).toBeInTheDocument();
  });

  it("shows a no-results state when every object search is empty", async () => {
    const user = userEvent.setup();
    renderPalette();
    await user.type(input(), "zzznomatch");
    await waitFor(() => expect(screen.getByText(/Nothing matches/)).toBeInTheDocument());
  });
});

describe("CommandPalette — partial failure", () => {
  it("keeps working when one object search fails", async () => {
    endpoints.getCustomers.mockResolvedValue(list([{ id: "c1", name: "Acme Corporation" }]));
    endpoints.getSubscriptions.mockRejectedValue(new Error("boom"));
    const user = userEvent.setup();
    renderPalette();
    await user.type(input(), "acme");
    await waitFor(() => expect(screen.getByText("Acme Corporation")).toBeInTheDocument());
    expect(screen.getByText(/Couldn.t search subscriptions/i)).toBeInTheDocument();
  });

  it("keeps the route launcher available when all object searches fail", async () => {
    endpoints.getCustomers.mockRejectedValue(new Error("x"));
    endpoints.getPlans.mockRejectedValue(new Error("x"));
    endpoints.getSubscriptions.mockRejectedValue(new Error("x"));
    const user = userEvent.setup();
    renderPalette();
    await user.type(input(), "customers");
    // The object search fails per group…
    await waitFor(() => expect(screen.getByText(/Couldn.t search customers/i)).toBeInTheDocument());
    // …but the route destination still resolves from the static nav list.
    expect(screen.getByText("Go to")).toBeInTheDocument();
  });
});

describe("CommandPalette — keyboard & navigation", () => {
  it("navigates to a customer result on Enter", async () => {
    endpoints.getCustomers.mockResolvedValue(list([{ id: "c1", name: "Acme Corporation" }]));
    const user = userEvent.setup();
    renderPalette();
    await user.type(input(), "acme");
    await waitFor(() => expect(screen.getByText("Acme Corporation")).toBeInTheDocument());
    await user.keyboard("{ArrowDown}{Enter}");
    expect(screen.getByTestId("loc")).toHaveTextContent("/customers/c1");
  });

  it("traverses across groups with Arrow keys", async () => {
    endpoints.getCustomers.mockResolvedValue(list([{ id: "c1", name: "Acme Corporation" }]));
    endpoints.getPlans.mockResolvedValue(list([{ id: "p1", name: "Acme Plan", code: "ACME" }]));
    const user = userEvent.setup();
    renderPalette();
    await user.type(input(), "acme");
    await waitFor(() => expect(screen.getByText("Acme Plan")).toBeInTheDocument());
    // Down twice: customer (index 0) → plan (index 1). Enter opens the plan.
    await user.keyboard("{ArrowDown}{ArrowDown}{Enter}");
    expect(screen.getByTestId("loc")).toHaveTextContent("/plans/p1");
  });

  it("closes on Escape", async () => {
    const onOpenChange = vi.fn();
    const user = userEvent.setup();
    renderPalette({ onOpenChange });
    await user.keyboard("{Escape}");
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("still runs the existing route search (regression)", async () => {
    const user = userEvent.setup();
    renderPalette();
    await user.type(input(), "subscriptions");
    // A nav destination matching the query is shown and navigable.
    const nav = await screen.findByText("Subscriptions");
    await user.click(nav);
    expect(screen.getByTestId("loc")).toHaveTextContent("/subscriptions");
  });
});

describe("CommandPalette — cancellation / stale isolation", () => {
  it("a late response for a previous query cannot overwrite the current query", async () => {
    // "ac" resolves slowly (300ms), "acm" resolves immediately.
    endpoints.getCustomers.mockImplementation(({ q }) =>
      new Promise((res) =>
        setTimeout(
          () => res(list([{ id: "k-" + q, name: "Result " + q }])),
          q === "ac" ? 300 : 0
        )
      )
    );
    const user = userEvent.setup();
    renderPalette();
    await user.type(input(), "ac");
    await new Promise((r) => setTimeout(r, 250)); // let the "ac" query fire
    await user.type(input(), "m"); // → "acm"
    await waitFor(() => expect(screen.getByText("Result acm")).toBeInTheDocument());
    // Wait past the slow "ac" resolution; it must NOT replace the current results.
    await new Promise((r) => setTimeout(r, 350));
    expect(screen.getByText("Result acm")).toBeInTheDocument();
    expect(screen.queryByText("Result ac")).not.toBeInTheDocument();
  });
});
