import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { CancelFlowStepConfig } from "../CancelFlowStepConfig";
import { endpoints } from "../../../lib/api";

vi.mock("../../../lib/api", () => ({
  endpoints: { getPlans: vi.fn() },
}));

beforeEach(() => {
  if (!Element.prototype.hasPointerCapture) Element.prototype.hasPointerCapture = () => false;
  if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = () => {};
});

const PLAN = { id: "p0000000-0000-0000-0000-000000000001", name: "Starter" };

const wrapper = ({ children }) => (
  <QueryClientProvider
    client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
  >
    {children}
  </QueryClientProvider>
);

describe("CancelFlowStepConfig", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getPlans.mockResolvedValue({ data: { data: [PLAN] } });
  });

  it("picks the plan-switch target from the shared plan list, not a pasted UUID", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <CancelFlowStepConfig
        stepType="offer"
        config={{ headline: "Wait!", offers: [{ type: "plan_switch" }] }}
        onChange={onChange}
      />,
      { wrapper }
    );
    expect(screen.queryByPlaceholderText(/uuid/i)).toBeNull();
    await waitFor(() => expect(endpoints.getPlans).toHaveBeenCalledWith({ limit: 1000 }));

    await user.click(document.getElementById("switch-to-plan-id"));
    await user.click(await screen.findByRole("option", { name: "Starter" }));

    expect(onChange).toHaveBeenCalledWith({
      headline: "Wait!",
      offers: [{ type: "plan_switch", switch_to_plan_id: PLAN.id }],
    });
  });

  it("edits survey reasons one per line", async () => {
    const onChange = vi.fn();
    render(<CancelFlowStepConfig stepType="survey" config={{}} onChange={onChange} />, { wrapper });
    // Controlled by the parent, so drive one change event with the full text.
    fireEvent.change(screen.getByLabelText("Reasons (one per line)"), {
      target: { value: "Too expensive\n\nMissing features " },
    });
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ questions: ["Too expensive", "Missing features"] })
    );
  });
});
