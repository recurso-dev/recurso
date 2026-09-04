import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import CreateQuote from "../CreateQuote";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getCustomers: vi.fn(),
    getQuote: vi.fn(),
    createQuote: vi.fn(),
    updateQuote: vi.fn(),
  },
}));

// jsdom lacks these; Radix (Sheet/Select) touches them.
beforeEach(() => {
  if (!Element.prototype.hasPointerCapture) Element.prototype.hasPointerCapture = () => false;
  if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = () => {};
});

const navigate = vi.fn();
vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal();
  return { ...actual, useNavigate: () => navigate };
});

const CUSTOMER = { id: "c0000000-0000-0000-0000-000000000001", name: "Acme Corp", email: "ops@acme.com" };

const renderAt = (path) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/quotes/new" element={<CreateQuote />} />
          <Route path="/quotes/:id/edit" element={<CreateQuote />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );

const inputsIn = (container) => container.querySelectorAll("input");

describe("CreateQuote (Sheet form)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getCustomers.mockResolvedValue({ data: { data: [CUSTOMER] } });
    endpoints.createQuote.mockResolvedValue({ data: { data: { id: "q_1" } } });
    endpoints.updateQuote.mockResolvedValue({ data: { data: { id: "q_1" } } });
  });

  it("requires a customer before creating", async () => {
    renderAt("/quotes/new");
    expect(screen.getByRole("heading", { name: "Create quote" })).toBeInTheDocument();
    await waitFor(() => expect(endpoints.getCustomers).toHaveBeenCalled());
    fireEvent.submit(document.getElementById("create-quote-form"));
    await waitFor(() => expect(screen.getByText("Select a customer.")).toBeInTheDocument());
    expect(endpoints.createQuote).not.toHaveBeenCalled();
  });

  // TEST_BACKLOG P0: every money field is edited in major units and must be
  // sent in minor units — unit price, line amount, tax and discount alike.
  it("converts every money field to minor units and previews the total correctly", async () => {
    const user = userEvent.setup();
    renderAt("/quotes/new");
    await waitFor(() => expect(endpoints.getCustomers).toHaveBeenCalled());

    await user.click(document.getElementById("customer_id"));
    await user.click(await screen.findByRole("option", { name: /Acme Corp/ }));

    fireEvent.change(screen.getByPlaceholderText("Item description"), {
      target: { value: "Onboarding" },
    });
    const line = screen.getByPlaceholderText("Item description").closest(".grid");
    const [, qty, unitPrice] = inputsIn(line);
    fireEvent.change(qty, { target: { value: "3" } });
    fireEvent.change(unitPrice, { target: { value: "12.50" } });
    fireEvent.change(document.querySelector('input[name="tax_amount"]'), {
      target: { value: "1.00" },
    });
    fireEvent.change(document.querySelector('input[name="discount_amount"]'), {
      target: { value: "0.50" },
    });

    // Preview: 3 × $12.50 = $37.50, + $1.00 tax − $0.50 discount = $38.00.
    expect(screen.getAllByText("$37.50").length).toBeGreaterThan(0);
    expect(screen.getByText("$38.00")).toBeInTheDocument();

    fireEvent.submit(document.getElementById("create-quote-form"));
    await waitFor(() => expect(endpoints.createQuote).toHaveBeenCalledTimes(1));
    const payload = endpoints.createQuote.mock.calls[0][0];
    expect(payload).toMatchObject({
      customer_id: CUSTOMER.id,
      currency: "USD",
      tax_amount: 100,
      discount_amount: 50,
      valid_until: null,
    });
    expect(payload.line_items).toEqual([
      { description: "Onboarding", quantity: 3, unit_price: 1250, amount: 3750 },
    ]);
    await waitFor(() => expect(navigate).toHaveBeenCalledWith("/quotes"));
  });

  it("loads a draft for editing in major units and saves it back in minor units", async () => {
    endpoints.getQuote.mockResolvedValue({
      data: {
        data: {
          id: "q_1",
          status: "draft",
          customer_id: CUSTOMER.id,
          currency: "USD",
          tax_amount: 200,
          discount_amount: 0,
          line_items: [{ description: "Seats", quantity: 2, unit_price: 5000 }],
        },
      },
    });
    renderAt("/quotes/q_1/edit");
    expect(screen.getByText("Edit quote")).toBeInTheDocument();
    // 5000 minor units renders as 50 in the editable field, not 5000.
    await waitFor(() => expect(screen.getByDisplayValue("Seats")).toBeInTheDocument());
    const line = screen.getByDisplayValue("Seats").closest(".grid");
    const [, qty, unitPrice] = inputsIn(line);
    expect(qty).toHaveValue(2);
    expect(unitPrice).toHaveValue(50);

    fireEvent.submit(document.getElementById("create-quote-form"));
    await waitFor(() => expect(endpoints.updateQuote).toHaveBeenCalledTimes(1));
    const [id, payload] = endpoints.updateQuote.mock.calls[0];
    expect(id).toBe("q_1");
    // Round-trips unchanged: no re-multiplication on save.
    expect(payload.tax_amount).toBe(200);
    expect(payload.line_items[0]).toMatchObject({ unit_price: 5000, amount: 10000 });
  });

  it("locks a quote that is no longer a draft", async () => {
    endpoints.getQuote.mockResolvedValue({
      data: { data: { id: "q_2", status: "sent", customer_id: CUSTOMER.id, currency: "USD", line_items: [] } },
    });
    renderAt("/quotes/q_2/edit");
    await waitFor(() =>
      expect(screen.getByText(/can no longer be edited/i)).toBeInTheDocument()
    );
    expect(screen.getByRole("button", { name: /save changes/i })).toBeDisabled();
    fireEvent.submit(document.getElementById("create-quote-form"));
    expect(endpoints.updateQuote).not.toHaveBeenCalled();
  });
});
