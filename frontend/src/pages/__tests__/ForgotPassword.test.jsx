import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach } from "vitest";
import ForgotPassword from "../ForgotPassword";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: { forgotPassword: vi.fn() },
}));

const renderPage = () =>
  render(
    <MemoryRouter>
      <ForgotPassword />
    </MemoryRouter>
  );

describe("ForgotPassword page", () => {
  beforeEach(() => vi.clearAllMocks());

  it("submits the email and shows the generic sent confirmation", async () => {
    endpoints.forgotPassword.mockResolvedValue({ data: {} });
    renderPage();
    const email = screen.getByPlaceholderText("you@company.com");
    fireEvent.change(email, { target: { value: "jane@acme.com" } });
    fireEvent.submit(email.closest("form"));
    await waitFor(() => expect(endpoints.forgotPassword).toHaveBeenCalledWith("jane@acme.com"));
    expect(screen.getByText(/If that account exists/i)).toBeInTheDocument();
  });

  it("shows the sent confirmation even if the request errors (no account enumeration)", async () => {
    endpoints.forgotPassword.mockRejectedValue(new Error("network"));
    renderPage();
    const email = screen.getByPlaceholderText("you@company.com");
    fireEvent.change(email, { target: { value: "jane@acme.com" } });
    fireEvent.submit(email.closest("form"));
    await waitFor(() => expect(screen.getByText(/If that account exists/i)).toBeInTheDocument());
  });
});
