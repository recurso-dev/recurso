import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Profile from "../Profile";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: { getAccount: vi.fn() },
}));

const navigate = vi.fn();
vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal();
  return { ...actual, useNavigate: () => navigate };
});

let authUser = { id: "u1", email: "ada@example.com", role: "owner" };
vi.mock("@/auth/AuthProvider", () => ({
  useAuth: () => ({ user: authUser }),
}));

const wrapper = ({ children }) => (
  <MemoryRouter>
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
    >
      {children}
    </QueryClientProvider>
  </MemoryRouter>
);

describe("Profile page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authUser = { id: "u1", email: "ada@example.com", role: "owner" };
  });

  it("shows the account identity and links session users to Security", async () => {
    endpoints.getAccount.mockResolvedValue({
      data: { data: { id: "ten_abc", name: "Acme Inc", email: "billing@acme.com" } },
    });
    render(<Profile />, { wrapper });
    expect(screen.getByText("Account profile")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText("Acme Inc")).toBeInTheDocument());
    expect(screen.getByText("billing@acme.com")).toBeInTheDocument();
    expect(screen.getByText("ten_abc")).toBeInTheDocument();
    expect(screen.getByText("Password & sessions")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /security settings/i }));
    expect(navigate).toHaveBeenCalledWith("/security");
  });

  it("describes API-key auth (and links to Developers) when there is no session user", async () => {
    authUser = null;
    endpoints.getAccount.mockResolvedValue({ data: { data: { id: "t", name: "N", email: "e" } } });
    render(<Profile />, { wrapper });
    await waitFor(() => expect(screen.getByText("API key authentication")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /manage keys/i }));
    expect(navigate).toHaveBeenCalledWith("/developers");
  });

  it("surfaces a load error without hiding the page", async () => {
    endpoints.getAccount.mockRejectedValue({
      response: { data: { error: { message: "account unavailable" } } },
    });
    render(<Profile />, { wrapper });
    expect(await screen.findByRole("alert")).toHaveTextContent("account unavailable");
  });
});
