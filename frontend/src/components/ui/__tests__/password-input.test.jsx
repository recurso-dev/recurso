import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { PasswordInput } from "../password-input";

describe("PasswordInput", () => {
  it("hides the value by default and toggles visibility", () => {
    render(<PasswordInput aria-label="Password" defaultValue="hunter2" />);
    const input = screen.getByLabelText("Password");
    expect(input).toHaveAttribute("type", "password");

    fireEvent.click(screen.getByRole("button", { name: /show password/i }));
    expect(input).toHaveAttribute("type", "text");

    fireEvent.click(screen.getByRole("button", { name: /hide password/i }));
    expect(input).toHaveAttribute("type", "password");
  });

  it("forwards props (id, autoComplete) through to the input", () => {
    render(<PasswordInput id="pw" autoComplete="new-password" aria-label="New password" />);
    const input = screen.getByLabelText("New password");
    expect(input).toHaveAttribute("id", "pw");
    expect(input).toHaveAttribute("autocomplete", "new-password");
  });
});
