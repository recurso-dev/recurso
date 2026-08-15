import { describe, it, expect, afterEach, vi } from "vitest";
import { render, screen, waitFor, act } from "@testing-library/react";

import { PaymentAttempts } from "../PaymentAttempts";

function setReducedMotion(matches) {
  window.matchMedia = vi.fn().mockImplementation((query) => ({
    matches,
    media: query,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }));
}

afterEach(() => {
  delete window.matchMedia;
});

const attempt = (over = {}) => ({
  id: "a1",
  status: "processing",
  method: "card",
  gateway: "stripe",
  amount: 118000,
  created_at: "2026-08-14T10:00:00Z",
  ...over,
});

describe("PaymentAttempts", () => {
  it("renders each attempt with its status", () => {
    render(<PaymentAttempts attempts={[attempt()]} currency="INR" />);
    expect(screen.getByText("processing")).toBeInTheDocument();
  });

  it("staggers the attempt list when motion is allowed", () => {
    setReducedMotion(false);
    render(
      <PaymentAttempts
        attempts={[attempt(), attempt({ id: "a2", status: "succeeded" })]}
        currency="INR"
      />
    );
    const items = screen.getAllByRole("listitem");
    expect(items[0]).toHaveClass("animate-motion-reveal");
    expect(items[1].style.animationDelay).toBe("50ms");
  });

  it("flashes the pill when an attempt's status advances", async () => {
    setReducedMotion(false);
    const { rerender } = render(
      <PaymentAttempts attempts={[attempt({ status: "processing" })]} currency="INR" />
    );
    // The flash lands on the MotionState wrapper — the pill span's parent.
    expect(screen.getByText("processing").parentElement).not.toHaveClass(
      "animate-motion-flash"
    );
    act(() => {
      rerender(
        <PaymentAttempts attempts={[attempt({ status: "succeeded" })]} currency="INR" />
      );
    });
    await waitFor(() =>
      expect(screen.getByText("succeeded").parentElement).toHaveClass(
        "animate-motion-flash"
      )
    );
  });

  it("leads with a humanized failure reason and keeps the raw code as detail", () => {
    render(
      <PaymentAttempts
        attempts={[attempt({ status: "failed", failure_code: "card_declined" })]}
        currency="INR"
      />
    );
    // Human-readable label is the primary, revealed reason.
    expect(screen.getByText("Card declined")).toHaveClass("animate-motion-reveal");
    // The raw gateway code is still present, but as quiet technical detail.
    expect(screen.getByText("card_declined")).toBeInTheDocument();
  });
});
