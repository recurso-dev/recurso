import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import CreatePlan from "../CreatePlan";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: { createPlan: vi.fn() },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const navigate = vi.fn();
vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal();
  return { ...actual, useNavigate: () => navigate };
});

const renderPage = () =>
  render(
    <MemoryRouter initialEntries={["/plans/new"]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <CreatePlan />
      </QueryClientProvider>
    </MemoryRouter>
  );

describe("CreatePlan (Sheet form)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.createPlan.mockResolvedValue({ data: { data: { id: "plan_1" } } });
  });

  it("refuses to submit without a name and code", async () => {
    renderPage();
    expect(screen.getByText("Create a new plan")).toBeInTheDocument();
    fireEvent.submit(document.getElementById("create-plan-form"));
    await waitFor(() => expect(screen.getByText("Plan name is required.")).toBeInTheDocument());
    expect(screen.getByText("Plan code is required.")).toBeInTheDocument();
    expect(endpoints.createPlan).not.toHaveBeenCalled();
  });

  // TEST_BACKLOG P0: the form edits in major units; the API contract is minor
  // units. $49.99 must reach the server as amount: 4999 — never 49.99.
  it("sends the price in minor units with the interval mapped to the API contract", async () => {
    renderPage();
    fireEvent.change(document.getElementById("name"), { target: { value: "Pro" } });
    fireEvent.change(document.getElementById("code"), { target: { value: "pro-yearly" } });
    fireEvent.change(document.getElementById("price"), { target: { value: "49.99" } });
    fireEvent.click(screen.getByRole("button", { name: "Yearly" }));
    fireEvent.submit(document.getElementById("create-plan-form"));

    await waitFor(() =>
      expect(endpoints.createPlan).toHaveBeenCalledWith({
        name: "Pro",
        code: "pro-yearly",
        currency: "USD",
        amount: 4999,
        interval_unit: "year",
        interval_count: 1,
      })
    );
    await waitFor(() => expect(navigate).toHaveBeenCalledWith("/plans"));
  });

  it("rejects a non-numeric price inline", async () => {
    renderPage();
    fireEvent.change(document.getElementById("name"), { target: { value: "Pro" } });
    fireEvent.change(document.getElementById("code"), { target: { value: "pro" } });
    fireEvent.change(document.getElementById("price"), { target: { value: "" } });
    fireEvent.submit(document.getElementById("create-plan-form"));
    await waitFor(() => expect(screen.getByText("Enter a valid price.")).toBeInTheDocument());
    expect(endpoints.createPlan).not.toHaveBeenCalled();
  });
});
