import { describe, it, expect, afterEach, vi } from "vitest";
import { render, screen } from "@testing-library/react";

import { JournalEntries } from "../JournalEntries";

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

const entries = [
  {
    transaction_id: "t1",
    code: 1,
    description: "Issuance",
    debit_account_code: "1100",
    debit_account_name: "Accounts Receivable",
    credit_account_code: "2100",
    credit_account_name: "Deferred Revenue",
    amount: 118000,
  },
  {
    transaction_id: "t2",
    code: 6,
    description: "Tax reclass",
    debit_account_code: "2100",
    debit_account_name: "Deferred Revenue",
    credit_account_code: "2200",
    credit_account_name: "Tax Payable",
    amount: 18000,
  },
];

describe("JournalEntries", () => {
  it("renders each posting and the balanced footer", () => {
    render(<JournalEntries entries={entries} currency="INR" />);
    expect(screen.getByText("Debits = Credits")).toBeInTheDocument();
    expect(screen.getAllByRole("listitem")).toHaveLength(2);
  });

  it("reveals postings in sequence when motion is allowed", () => {
    setReducedMotion(false);
    render(<JournalEntries entries={entries} currency="INR" />);
    const items = screen.getAllByRole("listitem");
    expect(items[0]).toHaveClass("animate-motion-reveal");
    expect(items[0].style.animationDelay).toBe("0ms");
    expect(items[1].style.animationDelay).toBe("55ms");
  });

  it("does not stagger under reduced motion", () => {
    setReducedMotion(true);
    render(<JournalEntries entries={entries} currency="INR" />);
    expect(screen.getAllByRole("listitem")[0]).not.toHaveClass(
      "animate-motion-reveal"
    );
  });

  it("shows the empty message with no postings", () => {
    render(<JournalEntries entries={[]} emptyMessage="Nothing posted yet" />);
    expect(screen.getByText("Nothing posted yet")).toBeInTheDocument();
  });
});
