import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import MCPSettings from "../MCPSettings";
import { endpoints } from "../../../lib/api";
import { toast } from "../../../components/ui/sonner";

vi.mock("../../../lib/api", () => ({
  endpoints: { getMCPSettings: vi.fn(), updateMCPSettings: vi.fn() },
}));
vi.mock("../../../components/ui/sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const wrapper = ({ children }) => (
  <QueryClientProvider
    client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
  >
    {children}
  </QueryClientProvider>
);

describe("MCPSettings", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.updateMCPSettings.mockResolvedValue({ data: { data: {} } });
  });

  it("loads the saved state with money-path tools off by default", async () => {
    endpoints.getMCPSettings.mockResolvedValue({ data: { data: { tier3_enabled: false } } });
    render(<MCPSettings />, { wrapper });
    expect(screen.getByText("MCP server")).toBeInTheDocument();
    const toggle = await screen.findByRole("switch", { name: /allow money-path tools/i });
    expect(toggle).toHaveAttribute("aria-checked", "false");
    expect(screen.getByText(/Issue a credit note \/ refund/)).toBeInTheDocument();
  });

  it("asks for confirmation before granting money-path tools, then saves", async () => {
    endpoints.getMCPSettings.mockResolvedValue({ data: { data: { tier3_enabled: false } } });
    render(<MCPSettings />, { wrapper });
    const toggle = await screen.findByRole("switch", { name: /allow money-path tools/i });
    fireEvent.click(toggle);
    fireEvent.click(screen.getByRole("button", { name: /save settings/i }));
    // High-consequence change: nothing is saved until confirmed.
    expect(await screen.findByText("Allow agents to move money?")).toBeInTheDocument();
    expect(endpoints.updateMCPSettings).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: /enable money-path tools/i }));
    await waitFor(() =>
      expect(endpoints.updateMCPSettings).toHaveBeenCalledWith({ tier3_enabled: true })
    );
    await waitFor(() => expect(toast.success).toHaveBeenCalled());
  });

  it("revokes money-path tools without a confirmation step", async () => {
    endpoints.getMCPSettings.mockResolvedValue({ data: { data: { tier3_enabled: true } } });
    render(<MCPSettings />, { wrapper });
    const toggle = await screen.findByRole("switch", { name: /allow money-path tools/i });
    await waitFor(() => expect(toggle).toHaveAttribute("aria-checked", "true"));
    fireEvent.click(toggle);
    fireEvent.click(screen.getByRole("button", { name: /save settings/i }));
    await waitFor(() =>
      expect(endpoints.updateMCPSettings).toHaveBeenCalledWith({ tier3_enabled: false })
    );
    expect(screen.queryByText("Allow agents to move money?")).toBeNull();
  });

  it("shows an error state with retry when settings fail to load", async () => {
    endpoints.getMCPSettings.mockRejectedValueOnce(new Error("boom"));
    endpoints.getMCPSettings.mockResolvedValue({ data: { data: { tier3_enabled: false } } });
    render(<MCPSettings />, { wrapper });
    expect(await screen.findByText("Couldn't load MCP settings")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    await waitFor(() => expect(endpoints.getMCPSettings).toHaveBeenCalledTimes(2));
    expect(await screen.findByText("Agent access")).toBeInTheDocument();
  });
});
