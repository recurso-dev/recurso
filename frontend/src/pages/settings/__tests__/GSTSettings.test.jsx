import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import GSTSettings from "../GSTSettings";
import { endpoints } from "../../../lib/api";
import { toast } from "../../../components/ui/sonner";

vi.mock("../../../lib/api", () => ({
  endpoints: {
    getGSTConfig: vi.fn(),
    validateGSTIN: vi.fn(),
    updateGSTConfig: vi.fn(),
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

const SAVED = {
  gstin: "",
  state_code: "",
  state_name: "",
  sac_code: "998314",
  gst_rate: 18,
  pan: "",
  legal_name: "Acme India Pvt Ltd",
  trade_name: "Acme",
  address: "1 MG Road",
  has_lut: false,
};

describe("GSTSettings", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getEntities.mockResolvedValue({ data: { data: [] } });
    endpoints.getGSTConfig.mockResolvedValue({ data: { data: SAVED } });
    endpoints.updateGSTConfig.mockResolvedValue({ data: { data: {} } });
  });

  it("rejects a short GSTIN locally, then auto-fills state and PAN from a valid one", async () => {
    endpoints.validateGSTIN.mockResolvedValue({
      data: { valid: true, message: "Valid GSTIN", state_code: "29", state_name: "Karnataka", pan: "ABCDE1234F" },
    });
    render(<GSTSettings />, { wrapper });
    expect(screen.getByText("GST configuration")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByLabelText("Legal name")).toHaveValue("Acme India Pvt Ltd"));

    fireEvent.change(screen.getByLabelText("GSTIN"), { target: { value: "29abc" } });
    fireEvent.click(screen.getByRole("button", { name: /^validate$/i }));
    expect(await screen.findByText("GSTIN must be 15 characters")).toBeInTheDocument();
    expect(endpoints.validateGSTIN).not.toHaveBeenCalled();

    fireEvent.change(screen.getByLabelText("GSTIN"), { target: { value: "29abcde1234f1z5" } });
    fireEvent.click(screen.getByRole("button", { name: /^validate$/i }));
    await waitFor(() => expect(endpoints.validateGSTIN).toHaveBeenCalledWith("29ABCDE1234F1Z5"));
    expect(await screen.findByText("Valid GSTIN")).toBeInTheDocument();
    expect(screen.getByLabelText("State code")).toHaveValue("29");
    expect(screen.getByLabelText("State name")).toHaveValue("Karnataka");
    expect(screen.getByLabelText("PAN")).toHaveValue("ABCDE1234F");
  });

  it("saves the full config including the LUT flag and numeric GST rate", async () => {
    render(<GSTSettings />, { wrapper });
    await waitFor(() => expect(screen.getByLabelText("Legal name")).toHaveValue("Acme India Pvt Ltd"));
    fireEvent.click(screen.getByLabelText(/LUT \(Letter of Undertaking\)/i));
    fireEvent.change(screen.getByLabelText("GST rate (%)"), { target: { value: "12" } });
    fireEvent.click(screen.getByRole("button", { name: /save configuration/i }));
    await waitFor(() =>
      expect(endpoints.updateGSTConfig).toHaveBeenCalledWith(
        expect.objectContaining({ has_lut: true, gst_rate: 12, sac_code: "998314" }),
        ""
      )
    );
    await waitFor(() => expect(toast.success).toHaveBeenCalled());
  });

  it("shows an error state with retry when the config fails to load", async () => {
    endpoints.getGSTConfig.mockRejectedValueOnce(new Error("down"));
    render(<GSTSettings />, { wrapper });
    expect(await screen.findByText("Couldn't load GST configuration")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    await waitFor(() => expect(endpoints.getGSTConfig).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.getByLabelText("Legal name")).toHaveValue("Acme India Pvt Ltd"));
  });
});
