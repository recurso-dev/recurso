import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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
    // jsdom lacks these; Radix Select touches them.
    if (!Element.prototype.hasPointerCapture) Element.prototype.hasPointerCapture = () => false;
    if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = () => {};
    endpoints.getUsers.mockResolvedValue({ data: { data: users } });
    endpoints.updateUserRole.mockResolvedValue({ data: {} });
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

  it("confirms before applying a role change (privilege change)", async () => {
    const user = userEvent.setup();
    render(<Team />, { wrapper });
    await waitFor(() => expect(screen.getByText("Member Bob")).toBeInTheDocument());

    // Promote Bob from member → admin via his row's role select.
    const bobRow = screen.getByText("bob@acme.com").closest("tr");
    await user.click(within(bobRow).getByRole("combobox"));
    await user.click(await screen.findByRole("option", { name: "admin" }));

    // A confirm step gates the change — nothing sent yet.
    expect(await screen.findByText(/change this teammate's role/i)).toBeInTheDocument();
    expect(endpoints.updateUserRole).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: /make admin/i }));
    await waitFor(() => expect(endpoints.updateUserRole).toHaveBeenCalledWith("u2", "admin"));
  });
});
