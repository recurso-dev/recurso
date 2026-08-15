import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect } from "vitest";

import { LedgerAccountLink } from "../LedgerAccountLink";

const wrap = (ui) => render(<MemoryRouter>{ui}</MemoryRouter>);

describe("LedgerAccountLink", () => {
  it("links to the account page when an id is present", () => {
    wrap(<LedgerAccountLink id="acc-123" label="Accounts Receivable (1100)" />);
    const link = screen.getByRole("link", { name: /accounts receivable/i });
    expect(link).toHaveAttribute("href", "/ledger/accounts/acc-123");
  });

  it("falls back to a short id when no label is given", () => {
    wrap(<LedgerAccountLink id="0123456789abcdef" />);
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/ledger/accounts/0123456789abcdef");
    expect(link).toHaveTextContent("01234567…");
  });

  it("renders the label as plain text (no link) when the id is absent", () => {
    // The invoice/credit-note journal rows carry the account name but not the id
    // yet (BACKEND GAP) — they must still show the account, just not link it.
    wrap(<LedgerAccountLink label="Deferred Revenue (2100)" />);
    expect(screen.queryByRole("link")).toBeNull();
    expect(screen.getByText("Deferred Revenue (2100)")).toBeInTheDocument();
  });

  it("treats the nil UUID as no link", () => {
    wrap(
      <LedgerAccountLink
        id="00000000-0000-0000-0000-000000000000"
        label="Suspense"
      />,
    );
    expect(screen.queryByRole("link")).toBeNull();
    expect(screen.getByText("Suspense")).toBeInTheDocument();
  });

  it("renders an em dash when there is neither id nor label", () => {
    wrap(<LedgerAccountLink />);
    expect(screen.queryByRole("link")).toBeNull();
    expect(screen.getByText("—")).toBeInTheDocument();
  });
});
