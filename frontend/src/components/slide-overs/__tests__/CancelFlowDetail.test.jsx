import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import CancelFlowDetail from "../CancelFlowDetail";
import { endpoints as api } from "../../../lib/api";

vi.mock("../../../lib/api", () => ({
  endpoints: {
    getCancelFlow: vi.fn(),
    getCancelFlowStats: vi.fn(),
    updateCancelFlow: vi.fn(),
    createCancelFlowStep: vi.fn(),
    updateCancelFlowStep: vi.fn(),
    deleteCancelFlowStep: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const flow = {
  id: "flow_1",
  name: "Save-offer flow",
  is_active: true,
  steps: [
    { id: "s1", step_order: 1, step_type: "survey", config: { questions: ["Too expensive"] } },
    { id: "s2", step_order: 2, step_type: "offer", config: { headline: "20% off" } },
  ],
};

const renderDetail = () =>
  render(<CancelFlowDetail flowId="flow_1" isOpen onClose={() => {}} />);

describe("CancelFlowDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.getCancelFlow.mockResolvedValue({ data: flow });
    api.getCancelFlowStats.mockResolvedValue({ data: {} });
    api.updateCancelFlow.mockResolvedValue({ data: {} });
  });

  it("loads and renders the flow with its steps", async () => {
    renderDetail();
    await waitFor(() => expect(screen.getByText("Save-offer flow")).toBeInTheDocument());
    // The offer step's headline is surfaced in its summary.
    expect(screen.getByText(/20% off/)).toBeInTheDocument();
    // An active flow offers the Deactivate control.
    expect(screen.getByRole("button", { name: /deactivate/i })).toBeInTheDocument();
  });

  it("deactivates an active flow", async () => {
    renderDetail();
    await waitFor(() => expect(screen.getByText("Save-offer flow")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /deactivate/i }));
    await waitFor(() =>
      expect(api.updateCancelFlow).toHaveBeenCalledWith("flow_1", { is_active: false })
    );
  });
});
