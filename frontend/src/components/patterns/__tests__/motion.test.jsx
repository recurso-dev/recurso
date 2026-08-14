import { describe, it, expect, afterEach, vi } from "vitest";
import { render, screen, waitFor, act } from "@testing-library/react";

import { MotionNumber } from "../MotionNumber";
import { MotionReveal, MotionStagger } from "../MotionReveal";
import { MotionState } from "../MotionState";
import { useReducedMotion } from "@/lib/useReducedMotion";

// jsdom has no matchMedia; set/unset it to drive the reduced-motion preference.
function setReducedMotion(matches) {
  if (matches === undefined) {
    delete window.matchMedia;
    return;
  }
  window.matchMedia = vi.fn().mockImplementation((query) => ({
    matches,
    media: query,
    onchange: null,
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

function Probe() {
  return <span data-testid="probe">{String(useReducedMotion())}</span>;
}

describe("useReducedMotion", () => {
  it("defaults to reduced when matchMedia is unavailable", () => {
    setReducedMotion(undefined);
    render(<Probe />);
    expect(screen.getByTestId("probe")).toHaveTextContent("true");
  });

  it("reflects the media query match", () => {
    setReducedMotion(false);
    render(<Probe />);
    expect(screen.getByTestId("probe")).toHaveTextContent("false");
  });

  it("is true when the user prefers reduced motion", () => {
    setReducedMotion(true);
    render(<Probe />);
    expect(screen.getByTestId("probe")).toHaveTextContent("true");
  });
});

describe("MotionNumber", () => {
  it("renders the formatted value on mount (no count-up)", () => {
    setReducedMotion(false);
    render(<MotionNumber value={128420} format={(n) => `$${n.toLocaleString()}`} />);
    expect(screen.getByText("$128,420")).toBeInTheDocument();
  });

  it("snaps straight to the new value under reduced motion", () => {
    setReducedMotion(true);
    const { rerender } = render(<MotionNumber value={100} />);
    expect(screen.getByText("100")).toBeInTheDocument();
    rerender(<MotionNumber value={250} />);
    expect(screen.getByText("250")).toBeInTheDocument();
  });

  it("settles on the new value after animating", async () => {
    setReducedMotion(false);
    const { rerender } = render(<MotionNumber value={0} />);
    rerender(<MotionNumber value={500} />);
    await waitFor(() => expect(screen.getByText("500")).toBeInTheDocument());
  });

  it("renders non-numeric values as-is", () => {
    setReducedMotion(false);
    render(<MotionNumber value={undefined} format={() => "—"} />);
    expect(screen.getByText("—")).toBeInTheDocument();
  });
});

describe("MotionReveal / MotionStagger", () => {
  it("renders children with the reveal animation when motion is allowed", () => {
    setReducedMotion(false);
    render(<MotionReveal>hello</MotionReveal>);
    const el = screen.getByText("hello");
    expect(el).toHaveClass("animate-motion-reveal");
  });

  it("renders children without animation under reduced motion", () => {
    setReducedMotion(true);
    render(<MotionReveal>hello</MotionReveal>);
    const el = screen.getByText("hello");
    expect(el).not.toHaveClass("animate-motion-reveal");
  });

  it("staggers direct children by cloning the reveal onto each", () => {
    setReducedMotion(false);
    render(
      <MotionStagger step={50}>
        <div data-testid="a">A</div>
        <div data-testid="b">B</div>
      </MotionStagger>,
    );
    expect(screen.getByTestId("a")).toHaveClass("animate-motion-reveal");
    expect(screen.getByTestId("b")).toHaveClass("animate-motion-reveal");
    expect(screen.getByTestId("b").style.animationDelay).toBe("50ms");
  });

  it("leaves staggered children untouched under reduced motion", () => {
    setReducedMotion(true);
    render(
      <MotionStagger>
        <div data-testid="a">A</div>
      </MotionStagger>,
    );
    expect(screen.getByTestId("a")).not.toHaveClass("animate-motion-reveal");
  });
});

describe("MotionState", () => {
  it("does not flash on first mount", () => {
    setReducedMotion(false);
    render(
      <MotionState motionKey="pending" data-testid="s">
        badge
      </MotionState>,
    );
    expect(screen.getByTestId("s")).not.toHaveClass("animate-motion-flash");
  });

  it("flashes when the key changes", async () => {
    setReducedMotion(false);
    const { rerender } = render(
      <MotionState motionKey="pending" data-testid="s">
        badge
      </MotionState>,
    );
    act(() => {
      rerender(
        <MotionState motionKey="succeeded" data-testid="s">
          badge
        </MotionState>,
      );
    });
    await waitFor(() =>
      expect(screen.getByTestId("s")).toHaveClass("animate-motion-flash"),
    );
  });

  it("does not flash under reduced motion", () => {
    setReducedMotion(true);
    const { rerender } = render(
      <MotionState motionKey="pending" data-testid="s">
        badge
      </MotionState>,
    );
    rerender(
      <MotionState motionKey="succeeded" data-testid="s">
        badge
      </MotionState>,
    );
    expect(screen.getByTestId("s")).not.toHaveClass("animate-motion-flash");
  });
});
