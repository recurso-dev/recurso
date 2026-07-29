import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import GSTReturns from "../GSTReturns";
import { endpoints as api } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getGSTR1: vi.fn(),
    getGSTR3B: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/components/patterns/ReportScopeSelect", () => ({
  ReportScopeSelect: () => <div data-testid="scope-select" />,
}));

describe("GSTReturns page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.getGSTR1.mockResolvedValue({
      data: { data: { total_taxable_value: 500000, marker: "GSTR1_PAYLOAD" }, gov_schema: {} },
    });
    api.getGSTR3B.mockResolvedValue({ data: { data: {}, gov_schema: {} } });
  });

  it("renders a build-return action", () => {
    render(<GSTReturns />);
    expect(screen.getAllByRole("button", { name: /build return/i }).length).toBeGreaterThanOrEqual(1);
  });

  it("builds the GSTR-1 return and shows the result", async () => {
    render(<GSTReturns />);
    // The first card is GSTR-1.
    fireEvent.click(screen.getAllByRole("button", { name: /build return/i })[0]);
    await waitFor(() => expect(api.getGSTR1).toHaveBeenCalled());
    await waitFor(() => expect(screen.getByText(/GSTR1_PAYLOAD/)).toBeInTheDocument());
  });
});
