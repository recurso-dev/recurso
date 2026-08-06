import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { CodeSample } from "../code-sample";

const TABS = [
  { label: "cURL", code: "curl https://api.example/v1/customers" },
  { label: "Node", code: "await fetch('https://api.example/v1/customers')" },
];

describe("CodeSample", () => {
  beforeEach(() => {
    // jsdom has no clipboard by default.
    Object.assign(navigator, {
      clipboard: { writeText: vi.fn().mockResolvedValue(undefined) },
    });
  });

  it("shows the first tab's code and switches on tab click", () => {
    render(<CodeSample tabs={TABS} />);
    expect(screen.getByText(TABS[0].code)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: "Node" }));
    expect(screen.getByText(TABS[1].code)).toBeInTheDocument();
  });

  it("copies the active tab's code and confirms", async () => {
    render(<CodeSample tabs={TABS} />);
    fireEvent.click(screen.getByRole("tab", { name: "Node" }));
    fireEvent.click(screen.getByRole("button", { name: /copy code/i }));

    await waitFor(() =>
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(TABS[1].code)
    );
    expect(await screen.findByText("Copied")).toBeInTheDocument();
  });

  it("accepts a single code/label without tabs", () => {
    render(<CodeSample label="Shell" code="echo hi" />);
    expect(screen.getByRole("tab", { name: "Shell" })).toBeInTheDocument();
    expect(screen.getByText("echo hi")).toBeInTheDocument();
  });
});
