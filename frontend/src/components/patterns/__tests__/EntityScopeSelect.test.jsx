import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { EntityScopeSelect } from "../EntityScopeSelect";
import { endpoints } from "@/lib/api";

vi.mock("@/lib/api", () => ({ endpoints: { getEntities: vi.fn() } }));

const wrapper = ({ children }) => (
  <QueryClientProvider
    client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
  >
    {children}
  </QueryClientProvider>
);

describe("EntityScopeSelect", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders nothing for a single-entity tenant (only the primary)", async () => {
    endpoints.getEntities.mockResolvedValue({
      data: { data: [{ id: "1", name: "ACME", is_primary: true }] },
    });
    const { container } = render(<EntityScopeSelect value="" onChange={() => {}} />, { wrapper });
    await waitFor(() => expect(endpoints.getEntities).toHaveBeenCalled());
    // Nothing to scope: no control is shown.
    expect(container.querySelector('[role="combobox"]')).toBeNull();
    expect(screen.queryByText("Legal entity")).toBeNull();
  });

  it("shows a scope selector labelled by the primary when multiple entities exist", async () => {
    endpoints.getEntities.mockResolvedValue({
      data: {
        data: [
          { id: "1", name: "ACME Inc", is_primary: true },
          { id: "2", name: "ACME UK", is_primary: false },
        ],
      },
    });
    render(<EntityScopeSelect value="" onChange={() => {}} />, { wrapper });
    // The empty value maps to the primary entity, shown in the trigger.
    await screen.findByText("ACME Inc (primary)");
    expect(screen.getByText("Legal entity")).toBeInTheDocument();
  });
});
