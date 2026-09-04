import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach } from "vitest";
import AcceptInvite from "../AcceptInvite";
import { endpoints } from "../../lib/api";
import { toast } from "@/components/ui/sonner";

vi.mock("../../lib/api", () => ({
  endpoints: { resetPassword: vi.fn() },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const navigate = vi.fn();
vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal();
  return { ...actual, useNavigate: () => navigate };
});

const renderPage = (search = "?token=inv_123") =>
  render(
    <MemoryRouter initialEntries={[`/accept-invite${search}`]}>
      <AcceptInvite />
    </MemoryRouter>
  );

const fill = (password, confirm = password) => {
  fireEvent.change(document.getElementById("password"), { target: { value: password } });
  fireEvent.change(document.getElementById("confirm"), { target: { value: confirm } });
  fireEvent.click(screen.getByRole("button", { name: /set password/i }));
};

describe("AcceptInvite page", () => {
  beforeEach(() => vi.clearAllMocks());

  it("treats a missing token as an invalid invitation", () => {
    renderPage("");
    expect(screen.getByText("Welcome to Recurso")).toBeInTheDocument();
    expect(screen.getByText(/invitation link is invalid or has expired/i)).toBeInTheDocument();
    expect(document.getElementById("password")).toBeNull();
  });

  it("validates password length and confirmation before calling the API", async () => {
    renderPage();
    fill("short");
    expect(await screen.findByRole("alert")).toHaveTextContent(/at least 8 characters/i);
    fill("longenough1", "different");
    expect(await screen.findByRole("alert")).toHaveTextContent(/don't match/i);
    expect(endpoints.resetPassword).not.toHaveBeenCalled();
  });

  it("sets the password through the reset endpoint with the invite token", async () => {
    endpoints.resetPassword.mockResolvedValue({ data: {} });
    renderPage();
    fill("correct-horse-battery");
    await waitFor(() =>
      expect(endpoints.resetPassword).toHaveBeenCalledWith("inv_123", "correct-horse-battery")
    );
    await waitFor(() => expect(navigate).toHaveBeenCalledWith("/login"));
    expect(toast.success).toHaveBeenCalled();
  });

  it("switches to the invalid-token state when the server rejects the token", async () => {
    endpoints.resetPassword.mockRejectedValue({ response: { status: 400 } });
    renderPage();
    fill("correct-horse-battery");
    await waitFor(() =>
      expect(screen.getByText(/invitation link is invalid or has expired/i)).toBeInTheDocument()
    );
    expect(navigate).not.toHaveBeenCalled();
  });
});
