import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import PortalRedeem from "../PortalRedeem";

const wrapper = ({ children }) => <MemoryRouter>{children}</MemoryRouter>;

beforeEach(() => {
  global.fetch = vi.fn();
});
afterEach(() => vi.restoreAllMocks());

describe("PortalRedeem (money path)", () => {
  it("redeems a gift code and shows success", async () => {
    global.fetch.mockResolvedValue({ ok: true, status: 200, json: () => Promise.resolve({}) });
    render(<PortalRedeem />, { wrapper });

    fireEvent.change(screen.getByPlaceholderText(/GIFT-/i), { target: { value: "GIFT-ABCD1234" } });
    fireEvent.click(screen.getByRole("button", { name: /redeem gift/i }));

    await waitFor(() => expect(screen.getByText(/redeemed successfully/i)).toBeInTheDocument());
    const [url, opts] = global.fetch.mock.calls[0];
    expect(url).toMatch(/\/portal\/api\/redeem$/);
    expect(JSON.parse(opts.body)).toEqual({ code: "GIFT-ABCD1234" });
  });

  it("shows the error message on an invalid code", async () => {
    global.fetch.mockResolvedValue({
      ok: false,
      status: 400,
      json: () => Promise.resolve({ error: { message: "Invalid gift code" } }),
    });
    render(<PortalRedeem />, { wrapper });

    fireEvent.change(screen.getByPlaceholderText(/GIFT-/i), { target: { value: "GIFT-BAD" } });
    fireEvent.click(screen.getByRole("button", { name: /redeem gift/i }));

    await waitFor(() => expect(screen.getByText(/Invalid gift code/i)).toBeInTheDocument());
  });
});
