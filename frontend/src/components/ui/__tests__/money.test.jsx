import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { Money } from "../money";

// A matcher for the whole `.money` element's text content (the symbol is split
// into its own child span, so a plain getByText("$42.00") wouldn't match).
const moneyText = (text) => (_, el) =>
  el?.classList?.contains("money") && el.textContent === text;

describe("Money", () => {
  it("renders the currency symbol in its own muted span", () => {
    render(<Money amountMinor={4200} currency="USD" />);
    const el = screen.getByText(moneyText("$42.00"));
    expect(el).toHaveClass("money");
    const symbol = el.querySelector(".money-symbol");
    expect(symbol).not.toBeNull();
    expect(symbol.textContent).toBe("$");
  });

  it("is exponent-aware — JPY has no decimals, KWD has three", () => {
    render(<Money amountMinor={1000} currency="JPY" />);
    expect(screen.getByText(moneyText("¥1,000"))).toBeInTheDocument();

    render(<Money amountMinor={4200} currency="KWD" />);
    // Intl separates the KWD code from the number with a narrow no-break space.
    const kwd = screen.getByText((_, el) => el?.classList?.contains("money") && /KWD.?4\.200/.test(el.textContent));
    expect(kwd).toBeInTheDocument();
  });

  it("renders negative amounts with a leading minus", () => {
    render(<Money amountMinor={-2500} currency="USD" />);
    expect(screen.getByText(moneyText("-$25.00"))).toBeInTheDocument();
  });

  it("renders zero, null and undefined as the currency's zero", () => {
    const { rerender } = render(<Money amountMinor={0} currency="USD" />);
    expect(screen.getByText(moneyText("$0.00"))).toBeInTheDocument();
    rerender(<Money amountMinor={null} currency="USD" />);
    expect(screen.getByText(moneyText("$0.00"))).toBeInTheDocument();
    rerender(<Money amountMinor={undefined} currency="USD" />);
    expect(screen.getByText(moneyText("$0.00"))).toBeInTheDocument();
  });

  it("defaults currency to USD when omitted", () => {
    render(<Money amountMinor={150000} />);
    expect(screen.getByText(moneyText("$1,500.00"))).toBeInTheDocument();
  });

  it("applies each size in the vocabulary and defaults to md (no extra size class)", () => {
    const { rerender } = render(<Money amountMinor={100} currency="USD" size="sm" />);
    expect(screen.getByText(moneyText("$1.00"))).toHaveClass("text-xs");

    rerender(<Money amountMinor={100} currency="USD" size="lg" />);
    expect(screen.getByText(moneyText("$1.00"))).toHaveClass("text-lg", "font-semibold");

    rerender(<Money amountMinor={100} currency="USD" size="hero" />);
    expect(screen.getByText(moneyText("$1.00"))).toHaveClass("text-2xl", "font-semibold");

    // md (default) adds no text-size class — it inherits the ambient size, so
    // unsized callers are visually unchanged.
    rerender(<Money amountMinor={100} currency="USD" />);
    const md = screen.getByText(moneyText("$1.00"));
    expect(md.className).not.toMatch(/text-(xs|sm|lg|xl|2xl)/);
  });

  it("still accepts an extra className alongside a size", () => {
    render(<Money amountMinor={100} currency="USD" size="hero" className="text-destructive" />);
    const el = screen.getByText(moneyText("$1.00"));
    expect(el).toHaveClass("text-2xl", "text-destructive");
  });

  it("exposes the full amount as accessible text content", () => {
    render(<Money amountMinor={2399900} currency="USD" size="hero" />);
    // Screen readers read the concatenated text, not the split spans.
    expect(screen.getByText(moneyText("$23,999.00"))).toBeInTheDocument();
  });
});
