import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

import { Checkbox } from "../checkbox";
import ConsentCheckbox from "../ConsentCheckbox";

describe("Checkbox", () => {
  it("renders a native checkbox and toggles via onChange", () => {
    const onChange = vi.fn();
    render(<Checkbox checked={false} onChange={onChange} aria-label="agree" />);
    const box = screen.getByRole("checkbox", { name: "agree" });
    expect(box).toBeInTheDocument();
    fireEvent.click(box);
    expect(onChange).toHaveBeenCalledTimes(1);
  });

  it("uses the token-based accent style (not a hardcoded color)", () => {
    render(<Checkbox aria-label="x" />);
    expect(screen.getByRole("checkbox")).toHaveClass("accent-primary");
  });

  it("forwards disabled", () => {
    render(<Checkbox disabled aria-label="x" />);
    expect(screen.getByRole("checkbox")).toBeDisabled();
  });
});

describe("ConsentCheckbox", () => {
  it("renders the recurring-billing consent card", () => {
    render(
      <ConsentCheckbox type="recurring_billing" amount="$29.00" billingInterval="month" />,
    );
    expect(screen.getByText("Authorize Recurring Payments")).toBeInTheDocument();
    expect(screen.getByRole("checkbox")).toBeInTheDocument();
  });

  it("fires onConsentChange with the full consent payload when checked", () => {
    const onConsentChange = vi.fn();
    render(
      <ConsentCheckbox
        type="recurring_billing"
        amount="$29.00"
        billingInterval="month"
        planName="Pro"
        onConsentChange={onConsentChange}
      />,
    );
    fireEvent.click(screen.getByRole("checkbox"));
    expect(onConsentChange).toHaveBeenCalledTimes(1);
    const payload = onConsentChange.mock.calls[0][0];
    expect(payload.type).toBe("recurring_billing");
    expect(payload.granted).toBe(true);
    expect(payload.version).toBe("2024.01.1");
    // The stored legal text is preserved verbatim (references amount + plan).
    expect(payload.consentText).toContain("$29.00 per month");
    expect(payload.consentText).toContain("Pro plan");
  });

  it("marks the card as consented on check (token classes, no hardcoded hex)", () => {
    const { container } = render(<ConsentCheckbox type="terms_of_service" />);
    const label = container.querySelector("label");
    expect(label.className).not.toContain("#");
    fireEvent.click(screen.getByRole("checkbox"));
    expect(label.className).toContain("border-primary");
  });
});
