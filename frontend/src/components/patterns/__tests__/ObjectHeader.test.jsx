import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect } from "vitest";
import { ObjectHeader } from "../ObjectPage";
import { Money } from "@/components/ui/money";

const renderHeader = (props) =>
  render(
    <MemoryRouter>
      <ObjectHeader {...props} />
    </MemoryRouter>
  );

describe("ObjectHeader hero", () => {
  it("renders identity, status and the primary amount when given one", () => {
    renderHeader({
      kicker: "Payment",
      title: "Payment · INV-9",
      badge: <span>Returned</span>,
      amount: <Money amountMinor={2400000} currency="USD" size="hero" />,
      amountLabel: "returned by the bank",
      meta: <span>meta-here</span>,
    });
    expect(screen.getByRole("heading", { name: "Payment · INV-9" })).toBeInTheDocument();
    expect(screen.getByText("Payment")).toBeInTheDocument();
    // Hero amount rendered as Money at hero size.
    const money = screen.getByText(
      (_, el) => el?.classList?.contains("money") && el.textContent === "$24,000.00"
    );
    expect(money).toHaveClass("text-2xl");
    expect(screen.getByText("returned by the bank")).toBeInTheDocument();
  });

  it("omits the amount block entirely when no amount is given", () => {
    const { container } = renderHeader({ kicker: "Customer", title: "Acme Inc" });
    expect(container.querySelector(".money")).toBeNull();
    expect(screen.getByRole("heading", { name: "Acme Inc" })).toBeInTheDocument();
  });

  it("orders the amount before the secondary metadata (identity → amount → meta)", () => {
    renderHeader({
      title: "INV-1",
      amount: <Money amountMinor={100000} currency="USD" size="hero" />,
      meta: <span data-testid="meta">the-meta</span>,
    });
    const money = screen.getByText(
      (_, el) => el?.classList?.contains("money") && el.textContent === "$1,000.00"
    );
    const meta = screen.getByTestId("meta");
    // The primary financial fact precedes the quiet id/date metadata in the DOM.
    expect(money.compareDocumentPosition(meta) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });
});
