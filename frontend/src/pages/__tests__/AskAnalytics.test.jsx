import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import AskAnalytics from "../AskAnalytics";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: { askAnalytics: vi.fn() },
}));

// jsdom in this config doesn't expose localStorage; mirror PageSmoke's stub.
const store = {};
vi.stubGlobal("localStorage", {
  getItem: (k) => (k in store ? store[k] : null),
  setItem: (k, v) => {
    store[k] = String(v);
  },
  removeItem: (k) => {
    delete store[k];
  },
  clear: () => {
    for (const k in store) delete store[k];
  },
});

const renderPage = () => render(<AskAnalytics />);

describe("AskAnalytics", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it("shows the first-run example gallery before any question", () => {
    renderPage();
    // Hero input + grouped example gallery teaching the breadth of questions.
    expect(screen.getByPlaceholderText(/Ask anything about your billing data/i)).toBeInTheDocument();
    expect(screen.getByText("Collections")).toBeInTheDocument();
    expect(screen.getByText("Which customers churned last month?")).toBeInTheDocument();
    expect(screen.getByText(/every answer shows the SQL it ran/i)).toBeInTheDocument();
  });

  it("asks a question and keeps the answer as a persisted history entry", async () => {
    // All-string columns → renders as a table (no Tremor chart, which jsdom
    // can't lay out), so we can assert on the persisted thread cleanly.
    endpoints.askAnalytics.mockResolvedValue({
      data: {
        data: [
          { customer: "Acme", plan: "Pro" },
          { customer: "Beta", plan: "Free" },
        ],
        query: "SELECT customer, plan FROM ...",
      },
    });

    renderPage();
    fireEvent.change(screen.getByLabelText("Question"), {
      target: { value: "Customers and plans" },
    });
    fireEvent.click(screen.getByRole("button", { name: /ask/i }));

    await waitFor(() =>
      expect(screen.getByText("Customers and plans")).toBeInTheDocument()
    );
    // 2 rows in the result.
    expect(screen.getByText(/2 rows/)).toBeInTheDocument();
    // Persisted to localStorage so a reload keeps it. The write happens in a
    // useEffect keyed on the history state, so poll for it rather than reading
    // synchronously (the effect can flush after the render assertion — this was
    // a CI flake).
    await waitFor(() => {
      const stored = JSON.parse(localStorage.getItem("recurso.ask.history.v1") || "[]");
      expect(stored).toHaveLength(1);
      expect(stored[0].question).toBe("Customers and plans");
    });
  });

  it("formats NUMERIC columns that arrive as strings instead of dumping trailing zeros", async () => {
    // Postgres NUMERIC/DECIMAL sums reach the client as strings like
    // "234820.000000000000". A single row renders as a table (no chart), so we
    // can assert the cell is grouped and de-zeroed.
    endpoints.askAnalytics.mockResolvedValue({
      data: {
        data: [{ name: "Sirius Systems", total_revenue: "234820.000000000000" }],
        query: "SELECT name, SUM(amount) AS total_revenue FROM ...",
      },
    });

    renderPage();
    fireEvent.change(screen.getByLabelText("Question"), {
      target: { value: "Top customer by revenue" },
    });
    fireEvent.click(screen.getByRole("button", { name: /ask/i }));

    await waitFor(() => expect(screen.getByText("234,820")).toBeInTheDocument());
    expect(screen.queryByText(/234820\.0+/)).not.toBeInTheDocument();
  });

  it("restores history from localStorage on mount", () => {
    localStorage.setItem(
      "recurso.ask.history.v1",
      JSON.stringify([
        { id: "1", question: "Prior question", data: [{ n: 1 }], query: "SELECT 1", ts: 1 },
      ])
    );
    renderPage();
    expect(screen.getByText("Prior question")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /clear history/i })).toBeInTheDocument();
  });

  it("surfaces a friendly message when GenAI is not configured", async () => {
    endpoints.askAnalytics.mockRejectedValue({ response: { status: 503 } });
    renderPage();
    fireEvent.change(screen.getByLabelText("Question"), { target: { value: "x" } });
    fireEvent.click(screen.getByRole("button", { name: /ask/i }));
    await waitFor(() =>
      expect(screen.getByText(/isn't configured on this deployment/i)).toBeInTheDocument()
    );
  });
});
