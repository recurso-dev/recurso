import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Churn from "../Churn";
import { endpoints as api } from "../../lib/api";

const renderChurn = () => render(<Churn />, { wrapper: ({ children }) => <MemoryRouter>{children}</MemoryRouter> });

vi.mock("../../lib/api", () => ({
  endpoints: {
    getChurnAlerts: vi.fn(),
    getHighRiskCustomers: vi.fn(),
    getCustomers: vi.fn().mockResolvedValue({ data: { data: [] } }),
    acknowledgeChurnAlert: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

describe("Churn page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.getHighRiskCustomers.mockResolvedValue({
      data: { data: [{ customer_id: "cus_1", score: 82, risk_level: "high", model_version: "v2" }] },
    });
    api.getChurnAlerts.mockResolvedValue({
      data: { data: [{ id: "al_1", customer_id: "cus_1", score: 82 }] },
    });
    api.acknowledgeChurnAlert.mockResolvedValue({ data: {} });
  });

  it("renders high-risk customers with their score", async () => {
    renderChurn();
    await waitFor(() => expect(screen.getByText("82")).toBeInTheDocument());
    expect(screen.getByText("Risk score")).toBeInTheDocument();
  });

  it("acknowledges an alert", async () => {
    renderChurn();
    const ackBtn = await screen.findByRole("button", { name: /^acknowledge$/i });
    fireEvent.click(ackBtn);
    await waitFor(() => expect(api.acknowledgeChurnAlert).toHaveBeenCalledWith("al_1"));
  });

  it("shows the no-alerts state when there are none", async () => {
    api.getChurnAlerts.mockResolvedValue({ data: { data: [] } });
    renderChurn();
    await waitFor(() => expect(screen.getByText("No open churn alerts")).toBeInTheDocument());
  });
});
