import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import GSTReturns from "../GSTReturns";
import { endpoints as api } from "../../lib/api";
import { money } from "@/test/money";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getGSTR1: vi.fn(),
    getGSTR3B: vi.fn(),
  },
}));
vi.mock("@/components/patterns/ReportScopeSelect", () => ({
  ReportScopeSelect: () => <div data-testid="scope-select" />,
}));

const gstr1Fixture = {
  b2b: [
    {
      gstin: "27AAAAA0000A1Z5",
      invoices: [
        {
          invoice_number: "INV-1",
          date: "2026-07-05T00:00:00Z",
          place_of_supply: "27",
          taxable_value: 50000,
          igst: 0,
          cgst: 4500,
          sgst: 4500,
          rate: 18,
        },
      ],
    },
  ],
  b2cs: [
    { place_of_supply: "27", rate: 18, taxable_value: 20000, igst: 0, cgst: 1800, sgst: 1800 },
  ],
  cdnr: [],
  hsn_summary: [
    { hsn_code: "9983", taxable_value: 70000, igst: 0, cgst: 6300, sgst: 6300, invoice_count: 2 },
  ],
  total_taxable_value: 70000,
  total_igst: 0,
  total_cgst: 6300,
  total_sgst: 6300,
  invoice_count: 2,
  total_credit_taxable_value: 0,
  total_credit_igst: 0,
  total_credit_cgst: 0,
  total_credit_sgst: 0,
  credit_note_count: 0,
};

const gstr3bFixture = {
  outward_taxable: { taxable_value: 60000, igst: 0, cgst: 5400, sgst: 5400 },
  zero_rated: { taxable_value: 0, igst: 0, cgst: 0, sgst: 0 },
  nil_exempt: { taxable_value: 0, igst: 0, cgst: 0, sgst: 0 },
  inward_reverse_charge: { taxable_value: 0, igst: 0, cgst: 0, sgst: 0 },
  non_gst: { taxable_value: 0, igst: 0, cgst: 0, sgst: 0 },
  inter_state_unregistered: [{ place_of_supply: "29", taxable_value: 10000, igst: 1800 }],
  invoice_count: 3,
  credit_note_count: 1,
};

describe("GSTReturns page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.getGSTR1.mockResolvedValue({ data: { data: gstr1Fixture, gov_schema: {} } });
    api.getGSTR3B.mockResolvedValue({ data: { data: gstr3bFixture, gov_schema: {} } });
  });

  it("renders a build-return action", () => {
    render(<GSTReturns />);
    expect(
      screen.getAllByRole("button", { name: /build return/i }).length
    ).toBeGreaterThanOrEqual(1);
  });

  it("builds GSTR-1 and renders structured sections, not a JSON dump", async () => {
    render(<GSTReturns />);
    fireEvent.click(screen.getAllByRole("button", { name: /build return/i })[0]);
    await waitFor(() => expect(api.getGSTR1).toHaveBeenCalled());

    // Control totals with real money formatting (50000 paise = ₹500.00).
    expect(await screen.findByText("Control totals")).toBeInTheDocument();
    expect(screen.getByText("INV-1")).toBeInTheDocument();
    expect(screen.getAllByText("27AAAAA0000A1Z5").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText(money("₹500.00")).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/B2CS — supplies to unregistered buyers/)).toBeInTheDocument();
    expect(screen.getByText("No credit notes in this period.")).toBeInTheDocument();
    expect(screen.getByText("9983")).toBeInTheDocument();

    // The statutory filing must not render as a raw JSON dump.
    expect(document.querySelector("pre")).toBeNull();
  });

  it("shows an ErrorState with retry when the build fails", async () => {
    api.getGSTR1.mockRejectedValueOnce({
      response: { status: 500, data: { error: { message: "boom" } } },
    });
    render(<GSTReturns />);
    fireEvent.click(screen.getAllByRole("button", { name: /build return/i })[0]);

    expect(await screen.findByText("Couldn't build GSTR-1")).toBeInTheDocument();
    expect(screen.getByText("boom")).toBeInTheDocument();

    // Retry re-fetches and renders the sections.
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    await waitFor(() => expect(api.getGSTR1).toHaveBeenCalledTimes(2));
    expect(await screen.findByText("Control totals")).toBeInTheDocument();
  });

  it("builds GSTR-3B and renders Table 3.1 rows and the zero-sections note", async () => {
    const user = userEvent.setup();
    if (!Element.prototype.hasPointerCapture)
      Element.prototype.hasPointerCapture = () => false;
    if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = () => {};

    render(<GSTReturns />);
    await user.click(screen.getByRole("tab", { name: "GSTR-3B" }));
    await user.click(screen.getByRole("button", { name: /build return/i }));
    await waitFor(() => expect(api.getGSTR3B).toHaveBeenCalled());

    expect(
      await screen.findByText(/3\.1\(a\) Outward taxable supplies/)
    ).toBeInTheDocument();
    expect(screen.getAllByText(money("₹600.00")).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/Sections 3\.1\(b\)–\(e\) are reported as zero/)).toBeInTheDocument();
    // Table 3.2 row for inter-state unregistered supplies.
    expect(screen.getByText("29")).toBeInTheDocument();
    expect(screen.getByText(/Built from 3 invoices and 1 credit note in/)).toBeInTheDocument();
  });
});
