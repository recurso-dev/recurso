import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

import ErrorBoundary from "../ErrorBoundary";

function Boom({ error }) {
  throw error;
}

describe("ErrorBoundary", () => {
  let reload;
  let originalLocation;

  beforeEach(() => {
    originalLocation = window.location;
    reload = vi.fn();
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { reload },
    });
    // React logs caught render errors — silence the expected noise.
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    Object.defineProperty(window, "location", {
      configurable: true,
      value: originalLocation,
    });
    vi.restoreAllMocks();
  });

  it("renders children when there is no error", () => {
    render(
      <ErrorBoundary>
        <p>all good</p>
      </ErrorBoundary>,
    );
    expect(screen.getByText("all good")).toBeInTheDocument();
  });

  it("shows a design-system error state for a generic error (Try again)", () => {
    render(
      <ErrorBoundary>
        <Boom error={new Error("kaboom")} />
      </ErrorBoundary>,
    );
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    expect(screen.getByText("kaboom")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /try again/i }),
    ).toBeInTheDocument();
    expect(reload).not.toHaveBeenCalled();
  });

  it("recovers a stale-chunk error with a real reload", () => {
    const err = new Error(
      "Failed to fetch dynamically imported module: https://app.recurso.dev/assets/Subscriptions-FcLqR1jk.js",
    );
    render(
      <ErrorBoundary>
        <Boom error={err} />
      </ErrorBoundary>,
    );
    expect(screen.getByText("Update available")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /reload/i }));
    expect(reload).toHaveBeenCalledTimes(1);
  });
});
