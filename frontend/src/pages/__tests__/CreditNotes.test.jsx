import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { BrowserRouter, MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import CreditNotes from "../CreditNotes";
import { money } from "@/test/money";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: { getCreditNotes: vi.fn(), getCreditNote: vi.fn() },
}));
vi.mock("../../components/slide-overs/CreditNoteDetail", () => ({
  default: ({ creditNote }) =>
    creditNote ? <div data-testid="cn-detail">{creditNote.id}</div> : null,
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

  // Row activation navigates to /credit-notes/:id (URL-driven detail), so
  // the test mounts real routes and the sheet is fed by the single GET.
  it("opens the detail sheet on row click", async () => {
    endpoints.getCreditNote.mockResolvedValue({ data: { data: notes[0] } });
    render(
      <MemoryRouter initialEntries={["/credit-notes"]}>
        <QueryClientProvider
          client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
        >
          <Routes>
            <Route path="/credit-notes" element={<CreditNotes />} />
            <Route path="/credit-notes/:id" element={<CreditNotes />} />
          </Routes>
        </QueryClientProvider>
      </MemoryRouter>
    );
    await waitFor(() => expect(screen.getByText("CN-001")).toBeInTheDocument());
    fireEvent.click(screen.getByText("CN-001"));
    await waitFor(() => expect(screen.getByTestId("cn-detail")).toHaveTextContent("cn_1"));
    expect(endpoints.getCreditNote).toHaveBeenCalledWith("cn_1");
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
