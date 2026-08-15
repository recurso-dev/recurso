import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi } from "vitest";
import {
  ObjectPageSkeleton,
  ObjectNotFound,
  ObjectPageError,
} from "../ObjectPage";

const inRouter = (ui) => render(<MemoryRouter>{ui}</MemoryRouter>);

describe("ObjectPageSkeleton", () => {
  it("announces loading politely and is busy", () => {
    const { container } = render(<ObjectPageSkeleton />);
    const status = screen.getByRole("status");
    expect(status).toHaveAttribute("aria-busy", "true");
    expect(screen.getByText("Loading…")).toBeInTheDocument();
    // The decorative skeleton geometry is aria-hidden from the announcement.
    expect(container.querySelectorAll('[aria-hidden="true"]').length).toBeGreaterThan(0);
  });

  it("omits the rail column when hasRail is false", () => {
    const { container } = render(<ObjectPageSkeleton hasRail={false} />);
    // With no rail, the grid has one child (the main column) instead of two.
    const grid = container.querySelector(".grid");
    expect(grid.children.length).toBe(1);
  });
});

describe("ObjectNotFound", () => {
  it("shows a clear heading, safe message, a back link, and NO retry", () => {
    inRouter(
      <ObjectNotFound objectLabel="invoice" identifier="3b234b40" backTo="/invoices" backLabel="Invoices" />
    );
    expect(screen.getByRole("heading", { name: "Invoice not found" })).toBeInTheDocument();
    expect(screen.getByText(/doesn’t exist, or you may not have access/i)).toBeInTheDocument();
    // No tenant/security detail leaked.
    expect(screen.queryByText(/another workspace|tenant/i)).not.toBeInTheDocument();
    const back = screen.getByRole("link", { name: /Invoices/ });
    expect(back).toHaveAttribute("href", "/invoices");
    // Retrying a not-found can't help — no retry button.
    expect(screen.queryByRole("button", { name: /retry/i })).not.toBeInTheDocument();
  });
});

describe("ObjectPageError", () => {
  it("shows a retryable error with a safe message and a back link", () => {
    const onRetry = vi.fn();
    inRouter(
      <ObjectPageError
        objectLabel="payment"
        error={{ response: { status: 500, data: { error: { message: "db exploded" } } } }}
        onRetry={onRetry}
        backTo="/payments"
        backLabel="Payments"
      />
    );
    expect(screen.getByRole("heading", { name: /Couldn’t load this payment/ })).toBeInTheDocument();
    // 5xx detail is never surfaced.
    expect(screen.queryByText(/db exploded/)).not.toBeInTheDocument();
    expect(screen.getByText(/went wrong/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /retry/i })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Payments/ })).toHaveAttribute("href", "/payments");
  });

  it("preserves a safe known 4xx message", () => {
    inRouter(
      <ObjectPageError
        objectLabel="invoice"
        error={{ response: { status: 400, data: { error: { message: "invalid invoice id" } } } }}
        onRetry={() => {}}
      />
    );
    expect(screen.getByText("invalid invoice id")).toBeInTheDocument();
  });
});
