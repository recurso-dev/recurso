import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect } from "vitest";
import { StatCard } from "../StatCard";
import { TooltipProvider } from "@/components/ui/tooltip";

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

  it("colors a positive delta success-token and a negative delta destructive-token", () => {
    const { rerender } = render(<StatCard label="x" value="1" delta="+5%" deltaType="positive" />);
    expect(screen.getByText("+5%").className).toContain("text-success");
    rerender(<StatCard label="x" value="1" delta="-5%" deltaType="negative" />);
    expect(screen.getByText("-5%").className).toContain("text-destructive");
  });

  it("applies a danger tone to the value", () => {
    render(<StatCard label="Overdue" value="$1,000" tone="danger" />);
    expect(screen.getByText("$1,000").className).toContain("text-destructive");
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

  it("exposes an info trigger for `definition` on a non-link tile", () => {
    render(
      <TooltipProvider>
        <StatCard label="Churn" value="2.1%" definition="Canceled over active + canceled." />
      </TooltipProvider>
    );
    expect(
      screen.getByRole("button", { name: /what does churn mean/i })
    ).toBeInTheDocument();
  });

  it("renders a numeric value through MotionNumber with an optional formatter", () => {
    // Under jsdom (no matchMedia → reduced motion) MotionNumber snaps to the
    // final, formatted value — so the displayed text is deterministic.
    render(<StatCard label="Active" value={1234} />);
    expect(screen.getByText("1,234")).toBeInTheDocument();

    render(<StatCard label="MRR" value={128400} format={(n) => `$${(n / 100).toLocaleString()}`} />);
    expect(screen.getByText("$1,284")).toBeInTheDocument();
  });

  it("renders a pre-formatted string value verbatim (no motion)", () => {
    render(<StatCard label="Churn" value="2.1%" />);
    expect(screen.getByText("2.1%")).toBeInTheDocument();
  });

  it("falls back to a native title (no nested button) when the tile is a link", () => {
    render(
      <MemoryRouter>
        <StatCard label="Churn" value="2.1%" to="/churn" definition="Canceled over active + canceled." />
      </MemoryRouter>
    );
    // No interactive tooltip trigger nested inside the tile's <a>.
    expect(screen.queryByRole("button", { name: /what does churn mean/i })).not.toBeInTheDocument();
    expect(screen.getByText("Churn")).toHaveAttribute("title", "Canceled over active + canceled.");
  });
});
