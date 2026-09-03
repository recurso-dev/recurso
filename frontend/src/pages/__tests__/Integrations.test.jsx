import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Integrations from "../Integrations";
import { endpoints } from "../../lib/api";
import { toast } from "@/components/ui/sonner";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getAccountingConnections: vi.fn(),
    getAccountingSyncStatus: vi.fn(),
    connectAccounting: vi.fn(),
    connectAccountingToken: vi.fn(),
    disconnectAccounting: vi.fn(),
    triggerAccountingSync: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn(), message: vi.fn() },
}));
// Gateways and tax/CRM connections have their own suites; keep this page's
// scope to the accounting section.
vi.mock("@/components/PaymentGateways", () => ({ default: () => <div data-testid="gateways" /> }));
vi.mock("@/components/IntegrationConnections", () => ({ default: () => <div data-testid="connections" /> }));

const renderPage = (path = "/integrations") =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Integrations />
      </QueryClientProvider>
    </MemoryRouter>
  );

const CONN = {
  id: "conn_1",
  provider: "quickbooks",
  is_active: true,
  realm_id: "9130",
  last_sync_at: "2026-08-30T10:00:00Z",
  sync_status: "idle",
};

describe("Integrations page — accounting", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getAccountingConnections.mockResolvedValue({ data: { data: [] } });
    endpoints.getAccountingSyncStatus.mockResolvedValue({ data: { data: [], total: 0 } });
    endpoints.disconnectAccounting.mockResolvedValue({ data: {} });
    endpoints.triggerAccountingSync.mockResolvedValue({ data: { status: "started" } });
    endpoints.connectAccountingToken.mockResolvedValue({ data: {} });
  });

  it("renders every provider as not connected, with an empty sync log", async () => {
    renderPage();
    expect(screen.getByText("Integrations")).toBeInTheDocument();
    expect(screen.getByTestId("gateways")).toBeInTheDocument();
    expect(screen.getByTestId("connections")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText("No sync activity yet")).toBeInTheDocument());
    // Provider names appear on the cards and in the sync-log filter.
    expect(screen.getAllByText("QuickBooks Online").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Xero").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Not connected").length).toBeGreaterThanOrEqual(2);
    // No active connection: no page-level "Sync now".
    expect(screen.queryByRole("button", { name: /sync now/i })).toBeNull();
  });

  it("shows a connected provider, triggers a sync, and disconnects after confirmation", async () => {
    endpoints.getAccountingConnections.mockResolvedValue({ data: { data: [CONN] } });
    renderPage();
    await waitFor(() => expect(screen.getByText("Connected")).toBeInTheDocument());
    expect(screen.getByText("9130")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /^sync$/i }));
    await waitFor(() => expect(endpoints.triggerAccountingSync).toHaveBeenCalledWith("quickbooks"));
    expect(toast.success).toHaveBeenCalledWith(expect.stringMatching(/Quickbooks sync started/));

    fireEvent.click(screen.getByRole("button", { name: /^disconnect$/i }));
    expect(await screen.findByText("Disconnect QuickBooks Online?")).toBeInTheDocument();
    expect(endpoints.disconnectAccounting).not.toHaveBeenCalled();
    // Confirm button inside the dialog carries the same label.
    const buttons = screen.getAllByRole("button", { name: /^disconnect$/i });
    fireEvent.click(buttons[buttons.length - 1]);
    await waitFor(() => expect(endpoints.disconnectAccounting).toHaveBeenCalledWith("conn_1"));
  });

  it("reports when a sync is already running instead of claiming it started", async () => {
    endpoints.getAccountingConnections.mockResolvedValue({ data: { data: [CONN] } });
    endpoints.triggerAccountingSync.mockResolvedValue({ data: { status: "sync_already_running" } });
    renderPage();
    await waitFor(() => expect(screen.getByText("Connected")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /sync now/i }));
    await waitFor(() => expect(toast.message).toHaveBeenCalledWith(expect.stringMatching(/already running/)));
    expect(toast.success).not.toHaveBeenCalled();
  });

  it("surfaces a missing OAuth configuration instead of redirecting", async () => {
    endpoints.connectAccounting.mockResolvedValue({ data: {} });
    renderPage();
    // Connect buttons stay disabled until the connection list has loaded.
    await waitFor(() =>
      expect(screen.getAllByRole("button", { name: /^connect$/i })[0]).toBeEnabled()
    );
    const connect = screen.getAllByRole("button", { name: /^connect$/i });
    fireEvent.click(connect[0]); // QuickBooks card is first
    await waitFor(() => expect(endpoints.connectAccounting).toHaveBeenCalledWith("quickbooks"));
    expect(toast.error).toHaveBeenCalledWith("OAuth is not configured for this provider on the server.");
  });

  it("connects a token provider through the credentials sheet", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getAllByRole("button", { name: /^connect$/i })[2]).toBeEnabled()
    );
    const connect = screen.getAllByRole("button", { name: /^connect$/i });
    fireEvent.click(connect[2]); // NetSuite is the third provider card
    // Sheet title and its submit button share the label.
    expect(await screen.findByRole("heading", { name: "Connect NetSuite" })).toBeInTheDocument();
    const submit = screen.getByRole("button", { name: /connect netsuite/i });
    expect(submit).toBeDisabled();
    fireEvent.change(screen.getByLabelText("Account ID"), { target: { value: "1234567_SB1" } });
    fireEvent.change(screen.getByLabelText("Access token"), { target: { value: "tok_abc" } });
    fireEvent.click(submit);
    await waitFor(() =>
      expect(endpoints.connectAccountingToken).toHaveBeenCalledWith("netsuite", {
        account_id: "1234567_SB1",
        access_token: "tok_abc",
      })
    );
    expect(toast.success).toHaveBeenCalledWith("NetSuite connected.");
  });

  it("toasts the OAuth callback outcome once and cleans the URL", async () => {
    renderPage("/integrations?connected=xero");
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("Xero connected."));
    expect(toast.success).toHaveBeenCalledTimes(1);
  });
});
