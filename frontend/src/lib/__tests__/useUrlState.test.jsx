import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router";

import { useUrlState } from "../useUrlState";

function Probe() {
  const [page, setPage] = useUrlState("page", 1, { parse: Number });
  const [status, setStatus] = useUrlState("status", "all");
  const loc = useLocation();
  return (
    <div>
      <span data-testid="page">{page}</span>
      <span data-testid="status">{status}</span>
      <span data-testid="search">{loc.search}</span>
      <button onClick={() => setPage(3)}>page3</button>
      <button onClick={() => setPage((p) => p + 1)}>inc</button>
      <button onClick={() => setPage(1)}>page1</button>
      <button onClick={() => setStatus("active")}>active</button>
    </div>
  );
}

const renderAt = (route) =>
  render(
    <MemoryRouter initialEntries={[route]}>
      <Probe />
    </MemoryRouter>,
  );

describe("useUrlState", () => {
  it("returns defaults when the params are absent", () => {
    renderAt("/customers");
    expect(screen.getByTestId("page")).toHaveTextContent("1");
    expect(screen.getByTestId("status")).toHaveTextContent("all");
  });

  it("reads existing params (parsed)", () => {
    renderAt("/customers?page=3&status=active");
    expect(screen.getByTestId("page")).toHaveTextContent("3");
    expect(screen.getByTestId("status")).toHaveTextContent("active");
  });

  it("writes a non-default value to the URL", () => {
    renderAt("/customers");
    fireEvent.click(screen.getByText("active"));
    expect(screen.getByTestId("status")).toHaveTextContent("active");
    expect(screen.getByTestId("search").textContent).toContain("status=active");
  });

  it("supports functional updates", () => {
    renderAt("/customers?page=3");
    fireEvent.click(screen.getByText("inc"));
    expect(screen.getByTestId("page")).toHaveTextContent("4");
  });

  it("omits the default from the URL (clean back to page 1)", () => {
    renderAt("/customers?page=3");
    fireEvent.click(screen.getByText("page1"));
    expect(screen.getByTestId("page")).toHaveTextContent("1");
    expect(screen.getByTestId("search").textContent).not.toContain("page=");
  });
});
