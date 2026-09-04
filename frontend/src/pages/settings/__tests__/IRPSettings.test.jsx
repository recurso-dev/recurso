import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import IRPSettings from "../IRPSettings";
import { endpoints } from "../../../lib/api";

vi.mock("../../../lib/api", () => ({
  endpoints: {
    getIRPConfig: vi.fn(),
    updateIRPConfig: vi.fn(),
    testIRPConfig: vi.fn(),
    getEntities: vi.fn(),
  },
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

describe("IRPSettings", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getEntities.mockResolvedValue({ data: { data: [] } });
    endpoints.getIRPConfig.mockResolvedValue({
      data: { data: { environment: "sandbox", gstin: "33ABCDE1234F1Z5", client_id: "cid", is_enabled: true } },
    });
    endpoints.updateIRPConfig.mockResolvedValue({ data: { data: {} } });
  });

  it("loads the saved credentials and saves edits with the GSTIN upper-cased", async () => {
    render(<IRPSettings />, { wrapper });
    expect(screen.getByText("IRP settings")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByLabelText("GSTIN")).toHaveValue("33ABCDE1234F1Z5"));
    expect(screen.getByLabelText("Client ID")).toHaveValue("cid");
    expect(screen.getByRole("switch", { name: /enable e-invoicing/i })).toHaveAttribute(
      "aria-checked",
      "true"
    );

    fireEvent.change(screen.getByLabelText("GSTIN"), { target: { value: "29abcde1234f1z5" } });
    fireEvent.change(screen.getByLabelText("Username"), { target: { value: "nic-user" } });
    fireEvent.click(screen.getByRole("button", { name: /save configuration/i }));
    await waitFor(() =>
      expect(endpoints.updateIRPConfig).toHaveBeenCalledWith(
        expect.objectContaining({ gstin: "29ABCDE1234F1Z5", username: "nic-user", client_id: "cid" }),
        ""
      )
    );
  });

  it("reports the connection test result inline — success and failure", async () => {
    endpoints.testIRPConfig.mockResolvedValueOnce({
      data: { success: true, message: "Authenticated with NIC sandbox" },
    });
    render(<IRPSettings />, { wrapper });
    await waitFor(() => expect(screen.getByLabelText("GSTIN")).toHaveValue("33ABCDE1234F1Z5"));
    fireEvent.click(screen.getByRole("button", { name: /test connection/i }));
    expect(await screen.findByText("Authenticated with NIC sandbox")).toBeInTheDocument();
    expect(endpoints.testIRPConfig).toHaveBeenCalledWith("");

    endpoints.testIRPConfig.mockRejectedValueOnce({
      response: { data: { error: { message: "invalid client secret" } } },
    });
    fireEvent.click(screen.getByRole("button", { name: /test connection/i }));
    expect(await screen.findByText("invalid client secret")).toBeInTheDocument();
    expect(endpoints.updateIRPConfig).not.toHaveBeenCalled();
  });
  it("shows a retryable error instead of a blank saveable form when the fetch fails", async () => {
    endpoints.getIRPConfig.mockRejectedValueOnce(new Error("boom"));
    render(<IRPSettings />, { wrapper });
    await screen.findByText("Couldn't load IRP settings");
    expect(screen.queryByRole("button", { name: /save/i })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    await waitFor(() => expect(endpoints.getIRPConfig).toHaveBeenCalledTimes(2));
  });
});
