import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect } from "vitest";
import { StatCard } from "../StatCard";

describe("StatCard", () => {
  it("renders the label, value, and hint", () => {
    render(<StatCard label="MRR" value="$42,000" hint="vs. last month" />);
    expect(screen.getByText("MRR")).toBeInTheDocument();
    expect(screen.getByText("$42,000")).toBeInTheDocument();
    expect(screen.getByText("vs. last month")).toBeInTheDocument();
  });

  it("shows a skeleton and hides the value + delta while loading", () => {
    render(<StatCard label="MRR" value="$42,000" delta="+5%" loading />);
    expect(screen.queryByText("$42,000")).not.toBeInTheDocument();
    expect(screen.queryByText("+5%")).not.toBeInTheDocument();
  });

  it("colors a positive delta emerald and a negative delta red", () => {
    const { rerender } = render(<StatCard label="x" value="1" delta="+5%" deltaType="positive" />);
    expect(screen.getByText("+5%").className).toContain("text-emerald-600");
    rerender(<StatCard label="x" value="1" delta="-5%" deltaType="negative" />);
    expect(screen.getByText("-5%").className).toContain("text-red-600");
  });

  it("applies a danger tone to the value", () => {
    render(<StatCard label="Overdue" value="$1,000" tone="danger" />);
    expect(screen.getByText("$1,000").className).toContain("text-red-600");
  });

  it("wraps the tile in a keyboard-focusable link when `to` is set", () => {
    render(
      <MemoryRouter>
        <StatCard label="Customers" value="12" to="/customers" />
      </MemoryRouter>
    );
    const link = screen.getByRole("link", { name: /Customers: view details/i });
    expect(link).toHaveAttribute("href", "/customers");
  });

  it("is not a link without `to`", () => {
    render(<StatCard label="Customers" value="12" />);
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });
});
