import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";

import { SubscriptionRef } from "../SubscriptionRef";

vi.mock("@/lib/useCustomers", () => ({
  useSubscriptions: () => [
    { id: "sub-1", plan_id: "plan-1" },
    { id: "sub-2", plan_id: "plan-missing" },
  ],
  usePlans: () => ({ names: { "plan-1": "Scale plan" } }),
}));

const renderRef = (props) =>
  render(
    <MemoryRouter>
      <SubscriptionRef {...props} />
    </MemoryRouter>,
  );

describe("SubscriptionRef", () => {
  it("labels the link with the plan name and links to the subscription", () => {
    renderRef({ subscriptionId: "sub-1" });
    const link = screen.getByRole("link", { name: "Scale plan" });
    expect(link).toHaveAttribute("href", "/subscriptions/sub-1");
  });

  it("falls back to a short id (mono) when the plan can't be resolved", () => {
    renderRef({ subscriptionId: "sub-2" });
    // plan-missing has no name → short id fragment, still a link.
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/subscriptions/sub-2");
    expect(link).toHaveClass("font-mono");
    expect(link.textContent).toMatch(/^sub-2…$/);
  });

  it("renders a dash for a missing subscription id", () => {
    renderRef({ subscriptionId: null });
    expect(screen.getByText("—")).toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });
});
