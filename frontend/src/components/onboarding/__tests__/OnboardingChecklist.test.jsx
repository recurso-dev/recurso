import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi } from "vitest";
import OnboardingChecklist from "../OnboardingChecklist";

const list = (rows) => Promise.resolve({ data: { data: rows } });

vi.mock("@/lib/api", async (importOriginal) => ({
  ...(await importOriginal()),
  endpoints: {
    getGatewayConnections: vi.fn(() =>
      Promise.resolve({ data: { data: { connections: [{ id: "c1", provider: "stripe" }] } } })
    ),
    getPlans: vi.fn(() => list([])),
    getCustomers: vi.fn(() => list([])),
    getSubscriptions: vi.fn(() => list([])),
  },
}));

describe("OnboardingChecklist", () => {
  it("counts a connected gateway as done", async () => {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>
          <OnboardingChecklist />
        </MemoryRouter>
      </QueryClientProvider>
    );
    expect(await screen.findByText(/1 of 4 steps done/)).toBeInTheDocument();
    expect(screen.getByText(/next:/).textContent).toContain("Create your first plan");
  });
});
