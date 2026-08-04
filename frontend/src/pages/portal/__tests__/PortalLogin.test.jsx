import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import PortalLogin from "../PortalLogin";

const wrapper = ({ children }) => <MemoryRouter>{children}</MemoryRouter>;

beforeEach(() => {
  global.fetch = vi.fn();
});
afterEach(() => vi.restoreAllMocks());

describe("PortalLogin", () => {
  it("submits the email and shows the check-your-email success state", async () => {
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve({}) });
    render(<PortalLogin />, { wrapper });

    fireEvent.change(screen.getByPlaceholderText(/you@company.com/i), {
      target: { value: "ada@example.com" },
    });
    fireEvent.click(screen.getByRole("button", { name: /send login link/i }));

    await waitFor(() => expect(screen.getByText(/Check your email/i)).toBeInTheDocument());
    // The request went to the magic-link endpoint with the email in the body.
    const [url, opts] = global.fetch.mock.calls[0];
    expect(url).toMatch(/\/portal\/auth\/request$/);
    expect(JSON.parse(opts.body)).toEqual({ email: "ada@example.com" });
  });

  it("shows the server error when the request fails", async () => {
    global.fetch.mockResolvedValue({
      ok: false,
      json: () => Promise.resolve({ error: { message: "rate limited" } }),
    });
    render(<PortalLogin />, { wrapper });

    fireEvent.change(screen.getByPlaceholderText(/you@company.com/i), {
      target: { value: "ada@example.com" },
    });
    fireEvent.click(screen.getByRole("button", { name: /send login link/i }));

    await waitFor(() => expect(screen.getByText(/rate limited/i)).toBeInTheDocument());
    expect(screen.queryByText(/Check your email/i)).toBeNull();
  });
});
