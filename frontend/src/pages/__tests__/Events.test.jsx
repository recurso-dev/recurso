import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Events from "../Events";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getEvents: vi.fn(),
    getEventTypes: vi.fn(),
    getEventDeliveries: vi.fn(),
    redeliverEvent: vi.fn(),
  },
}));
import userEvent from "@testing-library/user-event";
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const wrapper = ({ children }) => (
  <BrowserRouter>
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
    >
      {children}
    </QueryClientProvider>
  </BrowserRouter>
);

const sampleEvent = {
  id: "evt_1",
  type: "invoice.created",
  object_type: "invoice",
  object_id: "inv_abc12345",
  data: { amount: 1000 },
  created_at: "2026-07-31T10:00:00Z",
};

describe("Events (webhook inspector)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getEvents.mockResolvedValue({ data: { data: [sampleEvent] } });
    endpoints.getEventTypes.mockResolvedValue({
      data: { data: ["invoice.created", "invoice.paid"] },
    });
    endpoints.getEventDeliveries.mockResolvedValue({ data: { data: [] } });
    endpoints.redeliverEvent.mockResolvedValue({ data: { queued: 2 } });
  });

  // The type filter must reach the server — filtering only the loaded page
  // hid matching events on other pages.
  it("sends the type filter to the server and resets to the first page", async () => {
    if (!Element.prototype.hasPointerCapture) Element.prototype.hasPointerCapture = () => false;
    if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = () => {};
    const user = userEvent.setup();

    render(<Events />, { wrapper });
    await waitFor(() => expect(screen.getByText("invoice.created")).toBeInTheDocument());

    await user.click(screen.getByRole("combobox", { name: /filter by event type/i }));
    await user.click(await screen.findByRole("option", { name: "invoice.paid" }));

    await waitFor(() =>
      expect(endpoints.getEvents).toHaveBeenCalledWith(
        expect.objectContaining({ type: "invoice.paid", offset: 0 })
      )
    );
  });

  it("lists events and requests the first page", async () => {
    render(<Events />, { wrapper });
    await waitFor(() =>
      expect(endpoints.getEvents).toHaveBeenCalledWith(
        expect.objectContaining({ limit: 100, offset: 0 })
      )
    );
    expect(await screen.findByText("invoice.created")).toBeInTheDocument();
  });

  it("opens the event, loads its deliveries, and redelivers it", async () => {
    render(<Events />, { wrapper });
    fireEvent.click(await screen.findByText("invoice.created"));

    // Opening the event fetches its deliveries and shows the payload.
    await waitFor(() => expect(endpoints.getEventDeliveries).toHaveBeenCalledWith("evt_1"));
    expect(
      await screen.findByText((c) => c.includes('"amount"'))
    ).toBeInTheDocument();

    // One-click redelivery hits the redeliver endpoint for this event.
    fireEvent.click(screen.getByRole("button", { name: /redeliver/i }));
    await waitFor(() => expect(endpoints.redeliverEvent).toHaveBeenCalledWith("evt_1"));
  });
});
