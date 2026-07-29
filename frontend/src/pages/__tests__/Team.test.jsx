import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Team from "../Team";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getUsers: vi.fn(),
    inviteUser: vi.fn(),
    updateUserRole: vi.fn(),
    deleteUser: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

let mockRole = "owner";
vi.mock("@/auth/AuthProvider", () => ({
  useAuth: () => ({ user: { id: "me", role: mockRole } }),
}));

const users = [
  { id: "u1", name: "Owner Jane", email: "jane@acme.com", role: "owner" },
  { id: "u2", name: "Member Bob", email: "bob@acme.com", role: "member" },
];

const wrapper = ({ children }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
    {children}
  </QueryClientProvider>
);

describe("Team page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockRole = "owner";
    endpoints.getUsers.mockResolvedValue({ data: { data: users } });
  });

  it("renders team members and their roles", async () => {
    render(<Team />, { wrapper });
    await waitFor(() => expect(screen.getByText("Owner Jane")).toBeInTheDocument());
    expect(screen.getByText("bob@acme.com")).toBeInTheDocument();
  });

  it("shows the Add-member action to an owner/admin", async () => {
    render(<Team />, { wrapper });
    await waitFor(() => expect(screen.getByText("Owner Jane")).toBeInTheDocument());
    expect(screen.getByRole("button", { name: /add member/i })).toBeInTheDocument();
  });

  it("hides Add-member from a plain member", async () => {
    mockRole = "member";
    render(<Team />, { wrapper });
    await waitFor(() => expect(screen.getByText("Owner Jane")).toBeInTheDocument());
    expect(screen.queryByRole("button", { name: /add member/i })).not.toBeInTheDocument();
  });
});
