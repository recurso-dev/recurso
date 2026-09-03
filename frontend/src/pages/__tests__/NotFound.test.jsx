import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect } from "vitest";
import NotFound from "../NotFound";

describe("NotFound page", () => {
  it("names the missing path and offers a way home", () => {
    render(
      <MemoryRouter initialEntries={["/customers/does-not-exist"]}>
        <NotFound />
      </MemoryRouter>
    );
    expect(screen.getByRole("heading", { name: "Page not found" })).toBeInTheDocument();
    // The broken path is echoed so the reader can report it.
    expect(screen.getByText("/customers/does-not-exist")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Go to Home" })).toHaveAttribute("href", "/");
  });
});
