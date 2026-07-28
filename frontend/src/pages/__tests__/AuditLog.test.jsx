import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { BrowserRouter } from "react-router";
import { describe, it, expect, vi, beforeEach } from "vitest";
import AuditLog from "../AuditLog";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: { getAuditLogs: vi.fn() },
}));

const wrapper = ({ children }) => <BrowserRouter>{children}</BrowserRouter>;

describe("AuditLog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getAuditLogs.mockResolvedValue({ data: { data: [] } });
  });

  it("requests the first page with a limit and offset", async () => {
    render(<AuditLog />, { wrapper });
    await waitFor(() =>
      expect(endpoints.getAuditLogs).toHaveBeenCalledWith(
        expect.objectContaining({ limit: 100, offset: 0 })
      )
    );
  });

  it("passes a date range as RFC3339 from/to", async () => {
    render(<AuditLog />, { wrapper });
    await waitFor(() => expect(endpoints.getAuditLogs).toHaveBeenCalled());

    fireEvent.change(screen.getByLabelText("From date"), { target: { value: "2026-02-01" } });

    await waitFor(() => {
      const lastCall = endpoints.getAuditLogs.mock.calls.at(-1)[0];
      // An RFC3339 instant (exact value is timezone-dependent), and the filter
      // change resets to the first page.
      expect(lastCall.from).toMatch(/^\d{4}-\d{2}-\d{2}T.*Z$/);
      expect(lastCall.offset).toBe(0);
    });
  });
});
