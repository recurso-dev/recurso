import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import CreditNoteDetail from "../CreditNoteDetail";
import { endpoints } from "../../../lib/api";

vi.mock("../../../lib/api", () => ({
  endpoints: {
    approveCreditNote: vi.fn(),
    rejectCreditNote: vi.fn(),
    voidCreditNote: vi.fn(),
    getCreditNotePdf: vi.fn(),
  },
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

let mockRole = "admin";
vi.mock("@/auth/AuthProvider", () => ({
  useAuth: () => ({ user: { role: mockRole } }),
}));

const base = {
  id: "cn_123",
  customer_id: "cus_1",
  amount: 7582,
  balance: 7582,
  currency: "USD",
  type: "adjustment",
  reason: "downgrade_proration",
  created_at: "2026-06-13T00:00:00Z",
};

const renderCN = (creditNote) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <CreditNoteDetail creditNote={creditNote} isOpen onClose={() => {}} />
    </QueryClientProvider>
  );

describe("CreditNoteDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockRole = "admin";
    endpoints.voidCreditNote.mockResolvedValue({ data: {} });
    endpoints.approveCreditNote.mockResolvedValue({ data: {} });
    endpoints.rejectCreditNote.mockResolvedValue({ data: {} });
  });

  it("renders the amount and balance", () => {
    renderCN({ ...base, status: "issued" });
    // Total amount and balance remaining both read $75.82 here.
    expect(screen.getAllByText("$75.82").length).toBeGreaterThanOrEqual(1);
  });

  it("shows approve/reject only for a pending credit note", () => {
    renderCN({ ...base, status: "pending_approval" });
    expect(screen.getByRole("button", { name: /approve/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /reject/i })).toBeInTheDocument();
  });

  it("offers Void for an issued adjustment with a balance", () => {
    renderCN({ ...base, status: "issued" });
    expect(screen.getByRole("button", { name: /^void$/i })).toBeInTheDocument();
  });

  it("does NOT offer Void for a refund note", () => {
    renderCN({ ...base, status: "issued", type: "refund" });
    expect(screen.queryByRole("button", { name: /^void$/i })).not.toBeInTheDocument();
  });

  it("does NOT offer Void when the balance is zero", () => {
    renderCN({ ...base, status: "issued", balance: 0 });
    expect(screen.queryByRole("button", { name: /^void$/i })).not.toBeInTheDocument();
  });

  it("hides Void from non-admins", () => {
    mockRole = "member";
    renderCN({ ...base, status: "issued" });
    expect(screen.queryByRole("button", { name: /^void$/i })).not.toBeInTheDocument();
  });

  it("voids only after confirmation", async () => {
    renderCN({ ...base, status: "issued" });
    fireEvent.click(screen.getByRole("button", { name: /^void$/i }));
    // Confirm dialog appears; the mutation hasn't fired yet.
    expect(endpoints.voidCreditNote).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: /void credit note/i }));
    await waitFor(() => expect(endpoints.voidCreditNote).toHaveBeenCalledWith("cn_123"));
  });
});
