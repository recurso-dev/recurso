import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";

import { Alert } from "../alert";
import { StatusBadge } from "../status-badge";
import { Textarea } from "../textarea";
import { CopyableId } from "../copyable-id";
import { formatDateTime } from "@/lib/utils";

describe("Alert (canonical tinted panel)", () => {
  it("renders title + body with the variant treatment and a status role", () => {
    render(
      <Alert variant="warning" title="Trial ends soon">
        Add a payment method.
      </Alert>
    );
    const el = screen.getByRole("status");
    expect(el.className).toContain("border-warning/25");
    expect(screen.getByText("Trial ends soon")).toBeInTheDocument();
    expect(screen.getByText("Add a payment method.")).toBeInTheDocument();
  });

  it("danger announces as an alert", () => {
    render(<Alert variant="danger">Payment failed.</Alert>);
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });

  it("falls back to info for unknown variants", () => {
    render(<Alert variant="sparkly">hello</Alert>);
    expect(screen.getByRole("status").className).toContain("border-info/25");
  });
});

describe("StatusBadge (the only sanctioned status rendering)", () => {
  it("humanizes snake_case and maps to the canonical variant", () => {
    render(<StatusBadge status="past_due" />);
    const el = screen.getByText("Past due");
    expect(el.className).toContain("text-destructive");
  });

  it("is case-insensitive (e-invoice statuses arrive uppercase)", () => {
    render(<StatusBadge status="GENERATED" />);
    expect(screen.getByText("Generated").className).toContain("text-success");
  });

  it("resolves domain collisions via kind", () => {
    const { rerender } = render(<StatusBadge status="open" />);
    expect(screen.getByText("Open").className).toContain("text-info");
    rerender(<StatusBadge status="open" kind="dispute" />);
    expect(screen.getByText("Open").className).toContain("text-warning");
  });

  it("renders unknown statuses as neutral, never raw", () => {
    render(<StatusBadge status="some_new_state" />);
    expect(screen.getByText("Some new state")).toBeInTheDocument();
  });

  it("renders nothing without a status", () => {
    const { container } = render(<StatusBadge status="" />);
    expect(container).toBeEmptyDOMElement();
  });
});

describe("Textarea", () => {
  it("carries the invalid state styling hooks and forwards props", () => {
    render(<Textarea aria-invalid="true" placeholder="Notes" rows={4} />);
    const el = screen.getByPlaceholderText("Notes");
    expect(el).toHaveAttribute("rows", "4");
    expect(el.className).toContain("aria-[invalid=true]:border-destructive");
  });
});

describe("CopyableId", () => {
  it("shows the short form, copies the full value, and confirms", async () => {
    const writeText = vi.fn().mockResolvedValue();
    Object.assign(navigator, { clipboard: { writeText } });
    render(<CopyableId value="0d9c1a2b-3456-7890-abcd-ef0123456789" label="invoice ID" />);

    expect(screen.getByText("0d9c1a2b…")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Copy invoice ID" }));
    expect(writeText).toHaveBeenCalledWith("0d9c1a2b-3456-7890-abcd-ef0123456789");
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Copied" })).toBeInTheDocument()
    );
  });

  it("renders an em-dash when there is no value", () => {
    render(<CopyableId value={null} />);
    expect(screen.getByText("—")).toBeInTheDocument();
  });
});

describe("formatDateTime", () => {
  it("renders the canonical en-US timestamp", () => {
    expect(formatDateTime("2026-08-12T09:41:00Z")).toMatch(
      /Aug 12, 2026, \d{1,2}:\d{2} (AM|PM)/
    );
  });
  it("em-dashes empty and invalid input", () => {
    expect(formatDateTime(null)).toBe("—");
    expect(formatDateTime("not-a-date")).toBe("—");
  });
});
