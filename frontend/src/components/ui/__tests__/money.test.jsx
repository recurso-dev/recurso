import { render } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { Money } from "../money";

describe("Money", () => {
  it("renders USD minor units with two decimals", () => {
    const { container } = render(<Money amountMinor={4200} currency="USD" />);
    expect(container.textContent).toBe("$42.00");
  });

  it("renders a zero-decimal currency without cents", () => {
    const { container } = render(<Money amountMinor={4200} currency="JPY" />);
    expect(container.textContent).toBe("¥4,200");
  });

  it("renders a 3-decimal currency correctly", () => {
    const { container } = render(<Money amountMinor={4200} currency="KWD" />);
    // KWD shows 3 decimals: 4200 minor = 4.200
    expect(container.textContent).toContain("4.200");
  });

  it("renders negatives", () => {
    const { container } = render(<Money amountMinor={-8200} currency="USD" />);
    expect(container.textContent).toBe("-$82.00");
  });

  it("puts the currency symbol in a .money-symbol span", () => {
    const { container } = render(<Money amountMinor={100} currency="USD" />);
    const sym = container.querySelector(".money-symbol");
    expect(sym).not.toBeNull();
    expect(sym.textContent).toBe("$");
  });

  it("carries the money class and any extra className", () => {
    const { container } = render(<Money amountMinor={100} className="text-destructive" />);
    const root = container.firstChild;
    expect(root.className).toContain("money");
    expect(root.className).toContain("text-destructive");
  });

  it("defaults a nullish amount to zero", () => {
    const { container } = render(<Money amountMinor={null} currency="USD" />);
    expect(container.textContent).toBe("$0.00");
  });
});
