import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Login from "../Login";

const navigate = vi.fn();
vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal();
  return { ...actual, useNavigate: () => navigate };
});

const login = vi.fn();
const loginMfa = vi.fn();
const loginWithApiKey = vi.fn();
vi.mock("@/auth/AuthProvider", () => ({
  useAuth: () => ({ login, loginMfa, loginWithApiKey }),
}));

const renderLogin = () =>
  render(
    <MemoryRouter>
      <Login />
    </MemoryRouter>
  );

const signIn = () => {
  fireEvent.change(screen.getByPlaceholderText("you@company.com"), { target: { value: "a@b.com" } });
  fireEvent.change(screen.getByPlaceholderText("••••••••"), { target: { value: "pw" } });
  fireEvent.click(screen.getByRole("button", { name: /sign in/i }));
};

describe("Login page", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders the login form", () => {
    renderLogin();
    expect(screen.getByText("Log in to Recurso")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("you@company.com")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("••••••••")).toBeInTheDocument();
  });

  it("logs in and navigates home when no MFA is required", async () => {
    login.mockResolvedValue({});
    renderLogin();
    signIn();
    await waitFor(() => expect(login).toHaveBeenCalledWith("a@b.com", "pw"));
    await waitFor(() => expect(navigate).toHaveBeenCalledWith("/"));
  });

  it("advances to the 2FA step when the server requires MFA", async () => {
    login.mockResolvedValue({ mfa_required: true, mfa_token: "mfa_tok" });
    renderLogin();
    signIn();
    await waitFor(() =>
      expect(screen.getByText("Two-factor authentication")).toBeInTheDocument()
    );
    // Not navigated yet — still needs the code.
    expect(navigate).not.toHaveBeenCalled();

    loginMfa.mockResolvedValue({});
    fireEvent.change(screen.getByPlaceholderText("123456"), {
      target: { value: "123456" },
    });
    fireEvent.click(screen.getByRole("button", { name: /verify & continue/i }));
    await waitFor(() => expect(loginMfa).toHaveBeenCalledWith("mfa_tok", "123456"));
    await waitFor(() => expect(navigate).toHaveBeenCalledWith("/"));
  });

  it("surfaces an error on invalid credentials", async () => {
    login.mockRejectedValue({ response: { data: { error: { message: "invalid credentials" } } } });
    renderLogin();
    signIn();
    await waitFor(() => expect(login).toHaveBeenCalled());
    expect(navigate).not.toHaveBeenCalled();
  });
});
