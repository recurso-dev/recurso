import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach } from "vitest";
import VerifyEmail from "../VerifyEmail";
import { endpoints } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  endpoints: { verifyEmail: vi.fn() },
}));

const refreshUser = vi.fn().mockResolvedValue(null);
vi.mock("@/auth/AuthProvider", () => ({
  useAuth: () => ({ refreshUser }),
}));

const navigate = vi.fn();
vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal();
  return { ...actual, useNavigate: () => navigate };
});

const renderAt = (route) =>
  render(
    <MemoryRouter initialEntries={[route]}>
      <VerifyEmail />
    </MemoryRouter>
  );

describe("VerifyEmail page", () => {
  beforeEach(() => vi.clearAllMocks());

  it("verifies the token and shows the success state, refreshing auth", async () => {
    endpoints.verifyEmail.mockResolvedValue({ data: {} });
    renderAt("/verify-email?token=tok_abc");

    await waitFor(() =>
      expect(endpoints.verifyEmail).toHaveBeenCalledWith("tok_abc")
    );
    expect(await screen.findByText(/your email is verified/i)).toBeInTheDocument();
    expect(refreshUser).toHaveBeenCalled();
  });

  it("shows the invalid state without calling the API when the token is missing", () => {
    renderAt("/verify-email");
    expect(screen.getByText(/invalid or has expired/i)).toBeInTheDocument();
    expect(endpoints.verifyEmail).not.toHaveBeenCalled();
  });

  it("shows the invalid state when the token is rejected", async () => {
    endpoints.verifyEmail.mockRejectedValue({ response: { status: 400 } });
    renderAt("/verify-email?token=bad");
    expect(await screen.findByText(/invalid or has expired/i)).toBeInTheDocument();
  });
});
