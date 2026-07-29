import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Register from "../Register";

const navigate = vi.fn();
vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal();
  return { ...actual, useNavigate: () => navigate };
});

const registerAccount = vi.fn();
vi.mock("@/auth/AuthProvider", () => ({
  useAuth: () => ({ registerAccount }),
}));

const renderRegister = () =>
  render(
    <MemoryRouter>
      <Register />
    </MemoryRouter>
  );

const fillAndSubmit = () => {
  // Set each field, then submit. With handleChange as a functional update,
  // these batched changes accumulate (they didn't before the fix).
  fireEvent.change(screen.getByPlaceholderText("Acme Corp"), { target: { name: "company_name", value: "Acme" } });
  fireEvent.change(screen.getByPlaceholderText("Jane Doe"), { target: { name: "name", value: "Jane" } });
  fireEvent.change(screen.getByPlaceholderText("name@company.com"), { target: { name: "email", value: "jane@acme.com" } });
  fireEvent.change(screen.getByPlaceholderText("At least 8 characters"), { target: { name: "password", value: "password1" } });
  fireEvent.submit(screen.getByPlaceholderText("Acme Corp").closest("form"));
};

describe("Register page", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders the registration form", () => {
    renderRegister();
    expect(screen.getByPlaceholderText("Acme Corp")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("name@company.com")).toBeInTheDocument();
  });

  it("registers a workspace and navigates home", async () => {
    registerAccount.mockResolvedValue({});
    renderRegister();
    fillAndSubmit();
    await waitFor(() =>
      expect(registerAccount).toHaveBeenCalledWith(
        expect.objectContaining({ company_name: "Acme", email: "jane@acme.com", password: "password1" })
      )
    );
    await waitFor(() => expect(navigate).toHaveBeenCalledWith("/"));
  });

  it("blocks a too-short password before calling the API", async () => {
    renderRegister();
    fireEvent.change(screen.getByPlaceholderText("Acme Corp"), { target: { name: "company_name", value: "Acme" } });
    fireEvent.change(screen.getByPlaceholderText("name@company.com"), { target: { name: "email", value: "jane@acme.com" } });
    fireEvent.change(screen.getByPlaceholderText("At least 8 characters"), { target: { name: "password", value: "short" } });
    fireEvent.submit(screen.getByPlaceholderText("Acme Corp").closest("form"));
    await waitFor(() => expect(screen.getByText(/at least 8 characters/i)).toBeInTheDocument());
    expect(registerAccount).not.toHaveBeenCalled();
  });

  it("surfaces a registration error without navigating", async () => {
    registerAccount.mockRejectedValue({
      response: { data: { error: { message: "email already registered" } } },
    });
    renderRegister();
    fillAndSubmit();
    await waitFor(() => expect(screen.getByText(/email already registered/i)).toBeInTheDocument());
    expect(navigate).not.toHaveBeenCalled();
  });
});
