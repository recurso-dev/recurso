import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach } from "vitest";
import ResetPassword from "../ResetPassword";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: { resetPassword: vi.fn() },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const navigate = vi.fn();
vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal();
  return { ...actual, useNavigate: () => navigate };
});

const renderAt = (route) =>
  render(
    <MemoryRouter initialEntries={[route]}>
      <ResetPassword />
    </MemoryRouter>
  );

describe("ResetPassword page", () => {
  beforeEach(() => vi.clearAllMocks());

  it("resets the password with the token and redirects to login", async () => {
    endpoints.resetPassword.mockResolvedValue({ data: {} });
    renderAt("/reset-password?token=tok_123");
    const [pw, confirm] = screen.getAllByPlaceholderText("••••••••");
    fireEvent.change(pw, { target: { value: "newpassword1" } });
    fireEvent.change(confirm, { target: { value: "newpassword1" } });
    fireEvent.submit(pw.closest("form"));
    await waitFor(() =>
      expect(endpoints.resetPassword).toHaveBeenCalledWith("tok_123", "newpassword1")
    );
    await waitFor(() => expect(navigate).toHaveBeenCalledWith("/login"));
  });

  it("shows an invalid-link state when the token is missing", () => {
    renderAt("/reset-password");
    expect(screen.getByText(/invalid or has expired/i)).toBeInTheDocument();
    expect(screen.queryByPlaceholderText("••••••••")).not.toBeInTheDocument();
  });
});
