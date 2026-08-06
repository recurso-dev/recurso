import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import CancelFlows from "../CancelFlows";
import { endpoints as api } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getCancelFlows: vi.fn(),
    createCancelFlow: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/components/slide-overs/CancelFlowDetail", () => ({ default: () => <div /> }));

const FLOWS = [
  { id: "cf-1", name: "Standard retention flow", is_default: true, is_active: true, cooldown_days: 30 },
];

const renderPage = () =>
  render(<CancelFlows />, {
    wrapper: ({ children }) => (
      <MemoryRouter>
        <QueryClientProvider
          client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
        >
          {children}
        </QueryClientProvider>
      </MemoryRouter>
    ),
  });

describe("CancelFlows page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.getCancelFlows.mockResolvedValue({ data: { data: FLOWS } });
    api.createCancelFlow.mockResolvedValue({ data: { id: "cf-2" } });
  });

  it("lists flows with default + active badges", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Standard retention flow")).toBeInTheDocument()
    );
    expect(screen.getByText("Default")).toBeInTheDocument();
    expect(screen.getByText("Active")).toBeInTheDocument();
  });

  it("shows a retryable error when the list fails", async () => {
    api.getCancelFlows.mockRejectedValue({
      response: { data: { error: { message: "flows unavailable" } } },
    });
    renderPage();
    await waitFor(() => expect(screen.getByText("flows unavailable")).toBeInTheDocument());
    expect(screen.getByRole("button", { name: /retry/i })).toBeInTheDocument();
  });

  it("creates a flow and refreshes the list", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Standard retention flow")).toBeInTheDocument()
    );

    fireEvent.click(screen.getByRole("button", { name: /new flow/i }));
    fireEvent.change(screen.getByPlaceholderText("Standard retention flow"), {
      target: { value: "Winback flow" },
    });
    fireEvent.click(screen.getByRole("button", { name: /create flow/i }));

    await waitFor(() =>
      expect(api.createCancelFlow).toHaveBeenCalledWith(
        expect.objectContaining({ name: "Winback flow", cooldown_days: 30 })
      )
    );
    // Successful create refetches the (invalidated) list.
    await waitFor(() => expect(api.getCancelFlows.mock.calls.length).toBeGreaterThan(1));
  });
});
