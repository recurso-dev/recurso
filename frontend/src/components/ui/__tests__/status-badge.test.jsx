import { describe, it, expect, afterEach, vi } from "vitest";
import { render, screen, waitFor, act } from "@testing-library/react";

import { StatusBadge } from "../status-badge";

// jsdom has no matchMedia; drive the reduced-motion preference.
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

describe("StatusBadge", () => {
  it("humanizes the status label", () => {
    render(<StatusBadge status="past_due" />);
    expect(screen.getByText("Past due")).toBeInTheDocument();
  });

  it("does not wrap in a flash container by default", () => {
    render(<StatusBadge status="paid" />);
    // No MotionState wrapper span carrying the flash affordance.
    const el = screen.getByText("Paid");
    expect(el.closest("span.rounded-md")).toBeNull();
  });

  it("flashes when the status changes with flashOnChange", async () => {
    setReducedMotion(false);
    const { rerender } = render(<StatusBadge status="pending" flashOnChange />);
    // No flash on first mount.
    expect(
      screen.getByText("Pending").closest("span")
    ).not.toHaveClass("animate-motion-flash");

    act(() => {
      rerender(<StatusBadge status="paid" flashOnChange />);
    });
    await waitFor(() =>
      expect(screen.getByText("Paid").closest("span")).toHaveClass(
        "animate-motion-flash"
      )
    );
  });

  it("does not flash under reduced motion", () => {
    setReducedMotion(true);
    const { rerender } = render(<StatusBadge status="pending" flashOnChange />);
    rerender(<StatusBadge status="paid" flashOnChange />);
    expect(
      screen.getByText("Paid").closest("span")
    ).not.toHaveClass("animate-motion-flash");
  });
});
