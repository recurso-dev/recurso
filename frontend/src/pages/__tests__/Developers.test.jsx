import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Developers from "../Developers";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getAPIKeys: vi.fn(),
    getWebhooks: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getEventTypes: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getEvents: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getEventDeliveries: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getWebhookDeliveries: vi.fn().mockResolvedValue({ data: { data: [] } }),
    createKey: vi.fn(),
    revokeKey: vi.fn(),
    createWebhook: vi.fn(),
    deleteWebhook: vi.fn(),
    setWebhookStatus: vi.fn(),
    redeliverEvent: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const keys = [{ id: "key_1", key_prefix: "sk_live_abcd", is_active: true, created_at: "2026-01-02T00:00:00Z" }];

const wrapper = ({ children }) => (
  <BrowserRouter>
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
      {children}
    </QueryClientProvider>
  </BrowserRouter>
);

describe("Developers page — API keys", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getAPIKeys.mockResolvedValue({ data: { data: keys } });
    endpoints.createKey.mockResolvedValue({ data: { data: { key_value: "sk_live_secret" } } });
    endpoints.revokeKey.mockResolvedValue({ data: {} });
  });

  it("lists existing API keys by prefix", async () => {
    render(<Developers />, { wrapper });
    await waitFor(() => expect(screen.getByText(/sk_live_abcd/)).toBeInTheDocument());
  });

  it("creates a new API key", async () => {
    render(<Developers />, { wrapper });
    await waitFor(() => expect(screen.getByText(/sk_live_abcd/)).toBeInTheDocument());
    fireEvent.click(screen.getAllByRole("button", { name: /create api key/i })[0]);
    await waitFor(() => expect(endpoints.createKey).toHaveBeenCalled());
  });

  it("revokes a key only after confirmation", async () => {
    render(<Developers />, { wrapper });
    await waitFor(() => expect(screen.getByText(/sk_live_abcd/)).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /^revoke$/i }));
    // Confirm dialog is up; nothing sent yet.
    expect(endpoints.revokeKey).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: /revoke key/i }));
    await waitFor(() => expect(endpoints.revokeKey).toHaveBeenCalledWith("key_1"));
  });
});
