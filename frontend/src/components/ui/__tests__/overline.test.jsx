import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { createRef } from "react";

import { Overline, OVERLINE_CLASS } from "../overline";

describe("Overline", () => {
  it("renders a <div> by default carrying the canonical token", () => {
    render(<Overline>Invoice</Overline>);
    const el = screen.getByText("Invoice");
    expect(el.tagName).toBe("DIV");
    // Canonical role tokens are present (the single deliberate style).
    for (const cls of OVERLINE_CLASS.split(" ")) {
      expect(el).toHaveClass(cls);
    }
    expect(el).toHaveClass("text-subtle");
    expect(el).toHaveClass("uppercase");
  });

  it("is polymorphic and preserves dt semantics inside a description list", () => {
    render(
      <dl>
        <Overline as="dt">Amount due</Overline>
        <dd>$99.00</dd>
      </dl>
    );
    const dt = screen.getByText("Amount due");
    expect(dt.tagName).toBe("DT");
    expect(dt).toHaveClass("uppercase");
  });

  it("preserves th semantics inside a table header", () => {
    render(
      <table>
        <thead>
          <tr>
            <Overline as="th" scope="col">Method</Overline>
          </tr>
        </thead>
      </table>
    );
    const th = screen.getByText("Method");
    expect(th.tagName).toBe("TH");
    expect(th).toHaveAttribute("scope", "col");
  });

  it("renders inline as a span when asked", () => {
    render(<Overline as="span">MRR</Overline>);
    expect(screen.getByText("MRR").tagName).toBe("SPAN");
  });

  it("merges a caller className for layout without dropping the role tokens", () => {
    render(<Overline className="mb-3">Details</Overline>);
    const el = screen.getByText("Details");
    expect(el).toHaveClass("mb-3");
    expect(el).toHaveClass("text-subtle");
  });

  it("forwards a ref to the rendered element", () => {
    const ref = createRef();
    render(<Overline ref={ref}>Ref</Overline>);
    expect(ref.current).toBeInstanceOf(HTMLElement);
    expect(ref.current.textContent).toBe("Ref");
  });
});
