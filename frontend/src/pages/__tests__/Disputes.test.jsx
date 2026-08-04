import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Disputes from "../Disputes";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getDisputes: vi.fn(),
    resolveDispute: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const openDispute = {
  id: "d-1",
  invoice_id: "inv-1",
  customer_id: "cus-1",
  reason: "Double charged",
  status: "open",
  created_at: "2026-02-01T00:00:00Z",
};

const renderPage = () => {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <Disputes />
      </MemoryRouter>
    </QueryClientProvider>
  );
};

describe("Disputes review", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getDisputes.mockResolvedValue({ data: { data: [openDispute] } });
    endpoints.resolveDispute.mockResolvedValue({ data: { status: "resolved" } });
  });

  const openReviewDialog = async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("Double charged")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /review/i }));
    await waitFor(() => expect(screen.getByText("Review dispute")).toBeInTheDocument());
  };

  it("accepts a dispute without a credit by default", async () => {
    await openReviewDialog();
    fireEvent.click(screen.getByRole("button", { name: /^accept$/i }));
    await waitFor(() =>
      expect(endpoints.resolveDispute).toHaveBeenCalledWith("d-1", {
        outcome: "accept",
        note: "",
      })
    );
  });

  it("issues a credit when the box is checked", async () => {
    await openReviewDialog();
    fireEvent.click(screen.getByRole("checkbox"));
    fireEvent.click(screen.getByRole("button", { name: /accept & credit/i }));
    await waitFor(() =>
      expect(endpoints.resolveDispute).toHaveBeenCalledWith("d-1", {
        outcome: "accept",
        note: "",
        issue_credit: true,
      })
    );
  });

  it("rejects a dispute", async () => {
    await openReviewDialog();
    fireEvent.click(screen.getByRole("button", { name: /reject/i }));
    await waitFor(() =>
      expect(endpoints.resolveDispute).toHaveBeenCalledWith("d-1", {
        outcome: "reject",
        note: "",
      })
    );
  });

  it("paginates: a full page enables Next, which requests the next offset (#9)", async () => {
    // 26 rows = PER_PAGE (25) + 1, the sentinel that means "there's a next page".
    const fullPage = Array.from({ length: 26 }, (_, i) => ({
      ...openDispute,
      id: `d-${i}`,
      reason: `Reason ${i}`,
    }));
    endpoints.getDisputes.mockResolvedValue({ data: { data: fullPage } });
    renderPage();

    await waitFor(() => expect(screen.getByText("Reason 0")).toBeInTheDocument());
    // Default filter "open" → status passed alongside offset 0; only PER_PAGE render.
    expect(endpoints.getDisputes).toHaveBeenLastCalledWith("open", { limit: 26, offset: 0 });
    expect(screen.queryByText("Reason 25")).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: /next/i }));
    await waitFor(() =>
      expect(endpoints.getDisputes).toHaveBeenLastCalledWith("open", { limit: 26, offset: 25 })
    );
  });
});
