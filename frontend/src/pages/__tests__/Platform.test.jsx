import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Platform from "../Platform";
import { endpoints } from "../../lib/api";
import { setFounderToken, clearFounderToken } from "../../lib/founderToken";

const renderPage = () =>
  render(<Platform />, {
    wrapper: ({ children }) => (
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        {children}
      </QueryClientProvider>
    ),
  });

vi.mock("../../lib/api", () => ({
  endpoints: { platformMetrics: vi.fn() },
}));

describe("Platform (founder operator view)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    clearFounderToken();
  });

  it("shows the token gate when no token is set", () => {
    renderPage();
    expect(screen.getByLabelText("Founder token")).toBeInTheDocument();
    expect(endpoints.platformMetrics).not.toHaveBeenCalled();
  });

  it("renders the charge dry-run once a token is entered", async () => {
    endpoints.platformMetrics.mockResolvedValue({
      data: {
        total_tenants: 3,
        signups_last_30d: 2,
        activated_tenants: 1,
        trials_expiring_7d: 0,
        cloud_charge_currency: "USD",
        cloud_charge_total_minor: 9900,
        cloud_charges: [
          {
            tenant_id: "t1",
            name: "Spotify",
            email: "billing@spotify.com",
            tracked_revenue_minor: 4000000,
            collected_volume_minor: 3000000,
            would_charge_minor: 9900,
            reason: "$99 monthly cap",
          },
        ],
      },
    });

    renderPage();
    fireEvent.change(screen.getByLabelText("Founder token"), {
      target: { value: "secret-token" },
    });
    fireEvent.click(screen.getByRole("button", { name: /view dashboard/i }));

    await waitFor(() => expect(screen.getByText("Spotify")).toBeInTheDocument());
    // the token was sent as the bearer credential
    expect(endpoints.platformMetrics).toHaveBeenCalledWith("secret-token");
    // its dry-run charge renders ($99.00)
    expect(screen.getAllByText("$99.00").length).toBeGreaterThan(0);
    expect(screen.getByText("$99 monthly cap")).toBeInTheDocument();
  });

  it("re-prompts for the token on a 401", async () => {
    setFounderToken("bad");
    endpoints.platformMetrics.mockRejectedValue({ response: { status: 401 } });
    renderPage();
    await waitFor(() =>
      expect(screen.getByText(/token was rejected/i)).toBeInTheDocument(),
    );
  });
});
