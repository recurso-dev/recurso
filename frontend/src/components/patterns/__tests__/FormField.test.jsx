import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";

import { FormField } from "../FormField";

describe("FormField", () => {
  it("renders no error region when valid", () => {
    render(
      <FormField label="Email" htmlFor="email">
        <input id="email" />
      </FormField>,
    );
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("reveals the validation error and marks the control invalid", () => {
    render(
      <FormField label="Email" htmlFor="email" error="Enter a valid email">
        <input id="email" />
      </FormField>,
    );
    const alert = screen.getByRole("alert");
    expect(alert).toHaveTextContent("Enter a valid email");
    // The error animates in (reduced motion neutralizes it via global CSS).
    expect(alert).toHaveClass("animate-motion-reveal");
    // a11y wiring is preserved alongside the motion.
    expect(screen.getByRole("textbox")).toHaveAttribute("aria-invalid", "true");
  });

  it("associates the label with the control when no htmlFor/id is given", () => {
    render(
      <FormField label="Email" description="Where receipts go">
        <input />
      </FormField>,
    );
    const input = screen.getByLabelText("Email");
    expect(input).toHaveAttribute("id");
    expect(input).toHaveAccessibleDescription("Where receipts go");
  });

  it("keeps the child's own id when the caller set one but no htmlFor", () => {
    render(
      <FormField label="Email">
        <input id="email" />
      </FormField>,
    );
    expect(screen.getByLabelText("Email")).toHaveAttribute("id", "email");
  });
});
