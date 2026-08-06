import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect } from "vitest";

import Landing from "../Landing";

const renderLanding = () =>
  render(
    <MemoryRouter>
      <Landing />
    </MemoryRouter>
  );

describe("Landing (signed-out front door)", () => {
  it("shows the accounting-first hero and proof points", () => {
    renderLanding();
    expect(
      screen.getByRole("heading", { name: /billing your accountant can trust/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: /a real double-entry ledger/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: /tax & revenue recognition/i })
    ).toBeInTheDocument();
  });

  it("routes visitors to create an account or log in", () => {
    renderLanding();
    const registerLinks = screen
      .getAllByRole("link")
      .filter((a) => a.getAttribute("href") === "/register");
    const loginLinks = screen
      .getAllByRole("link")
      .filter((a) => a.getAttribute("href") === "/login");
    expect(registerLinks.length).toBeGreaterThan(0);
    expect(loginLinks.length).toBeGreaterThan(0);
  });
});
