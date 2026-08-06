import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { CopyableSecret } from "../copyable-secret";

describe("CopyableSecret", () => {
  beforeEach(() => {
    Object.assign(navigator, {
      clipboard: { writeText: vi.fn().mockResolvedValue(undefined) },
    });
  });

  it("shows the value and copies it with confirmation", async () => {
    render(<CopyableSecret value="rsk_live_secret" ariaLabel="Secret API key" />);
    expect(screen.getByLabelText("Secret API key")).toHaveValue("rsk_live_secret");

    fireEvent.click(screen.getByRole("button", { name: /copy to clipboard/i }));
    await waitFor(() =>
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith("rsk_live_secret")
    );
    // Button now confirms the copy.
    expect(await screen.findByRole("button", { name: /copied to clipboard/i })).toBeInTheDocument();
  });

  it("masks the value until revealed when mask is set", () => {
    render(<CopyableSecret value="whsec_x" ariaLabel="Signing secret" mask />);
    const input = screen.getByLabelText("Signing secret");
    expect(input).toHaveAttribute("type", "password");

    fireEvent.click(screen.getByRole("button", { name: /reveal secret/i }));
    expect(input).toHaveAttribute("type", "text");
  });

  it("has no reveal toggle when unmasked", () => {
    render(<CopyableSecret value="visible" />);
    expect(screen.queryByRole("button", { name: /reveal secret/i })).not.toBeInTheDocument();
  });
});
