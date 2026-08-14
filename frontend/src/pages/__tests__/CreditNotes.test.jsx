import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import CreditNotes from "../CreditNotes";
import { money } from "@/test/money";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: { getCreditNotes: vi.fn() },
}));

const notes = [
  { id: "cn_1", reference: "CN-001", customer: { name: "Acme" }, amount: 5000, balance: 4000, currency: "USD", status: "issued" },
  { id: "cn_2", reference: "CN-002", customer: { name: "Beta" }, amount: 2500, balance: 0, currency: "USD", status: "used" },
];

const wrapper = ({ children }) => (
  <BrowserRouter>
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
      {children}
    </QueryClientProvider>
  </BrowserRouter>
);

describe("CreditNotes page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getCreditNotes.mockResolvedValue({ data: { data: notes } });
  });

  it("renders credit notes with reference and amount", async () => {
    render(<CreditNotes />, { wrapper });
    await waitFor(() => expect(screen.getByText("CN-001")).toBeInTheDocument());
    expect(screen.getByText("Acme")).toBeInTheDocument();
    expect(screen.getByText(money("$50.00"))).toBeInTheDocument();
  });

  // Rows link to the addressable credit-note object page (/credit-notes/:id).
  it("links each row to its credit-note page", async () => {
    render(<CreditNotes />, { wrapper });
    await waitFor(() => expect(screen.getByText("CN-001")).toBeInTheDocument());
    expect(screen.getByText("CN-001").closest("a")).toHaveAttribute(
      "href",
      "/credit-notes/cn_1"
    );
  });

  it("filters by customer name via search", async () => {
    render(<CreditNotes />, { wrapper });
    await waitFor(() => expect(screen.getByText("CN-001")).toBeInTheDocument());
    await userEvent.type(screen.getByPlaceholderText(/search/i), "Beta");
    expect(screen.queryByText("CN-001")).not.toBeInTheDocument();
    expect(screen.getByText("CN-002")).toBeInTheDocument();
  });

  it("shows the empty state with no credit notes", async () => {
    endpoints.getCreditNotes.mockResolvedValue({ data: { data: [] } });
    render(<CreditNotes />, { wrapper });
    await waitFor(() => expect(screen.getByText("No credit notes yet")).toBeInTheDocument());
  });
});
