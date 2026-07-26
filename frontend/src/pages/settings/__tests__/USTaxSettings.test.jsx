import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import USTaxSettings from "../USTaxSettings";
import { endpoints } from "../../../lib/api";

vi.mock("../../../lib/api", () => ({
  endpoints: { getUSTaxConfig: vi.fn(), updateUSTaxConfig: vi.fn() },
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

describe("USTaxSettings", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.updateUSTaxConfig.mockResolvedValue({ data: { data: {} } });
  });

  it("loads the saved W-9 identity and saves edits", async () => {
    endpoints.getUSTaxConfig.mockResolvedValue({
      data: { data: { legal_name: "Acme Inc", ein: "12-3456789", address: "1 Market St" } },
    });
    render(<USTaxSettings />, { wrapper });

    await waitFor(() => expect(screen.getByLabelText("Legal name")).toHaveValue("Acme Inc"));
    expect(screen.getByLabelText("EIN")).toHaveValue("12-3456789");

    fireEvent.click(screen.getByRole("button", { name: /save settings/i }));
    await waitFor(() =>
      expect(endpoints.updateUSTaxConfig).toHaveBeenCalledWith(
        expect.objectContaining({ ein: "12-3456789", legal_name: "Acme Inc" }),
      ),
    );
  });

  it("starts empty when no config is set", async () => {
    endpoints.getUSTaxConfig.mockResolvedValue({ data: { data: null } });
    render(<USTaxSettings />, { wrapper });
    await waitFor(() => expect(screen.getByLabelText("EIN")).toHaveValue(""));
  });
});
