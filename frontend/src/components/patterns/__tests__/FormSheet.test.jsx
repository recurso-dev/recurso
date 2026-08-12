import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { FormSheet } from "../FormSheet";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

function renderSheet(props = {}) {
  const onSubmit = vi.fn();
  const onOpenChange = vi.fn();
  render(
    <FormSheet
      open
      onOpenChange={onOpenChange}
      title="New thing"
      description="Creates a thing."
      onSubmit={onSubmit}
      submitLabel="Create thing"
      {...props}
    >
      <div>
        <Label htmlFor="name">Name</Label>
        <Input id="name" defaultValue="" />
      </div>
    </FormSheet>
  );
  return { onSubmit, onOpenChange };
}

describe("FormSheet", () => {
  it("submits on Enter — children live inside a real <form>", async () => {
    const { onSubmit } = renderSheet();
    fireEvent.submit(screen.getByRole("button", { name: "Create thing" }).closest("form"));
    expect(onSubmit).toHaveBeenCalled();
  });

  it("autofocuses the first field on open", async () => {
    renderSheet();
    await waitFor(() => expect(screen.getByLabelText("Name")).toHaveFocus());
  });

  it("blocks submit when canSubmit is false", () => {
    const { onSubmit } = renderSheet({ canSubmit: false });
    const btn = screen.getByRole("button", { name: "Create thing" });
    expect(btn).toBeDisabled();
    fireEvent.submit(btn.closest("form"));
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("closes without ceremony when the form is clean", () => {
    const { onOpenChange } = renderSheet({ dirty: false });
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("asks before discarding a dirty form, then closes on Discard", async () => {
    const { onOpenChange } = renderSheet({ dirty: true });
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    // No silent discard: the sheet is still open, a confirm intervenes.
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
    expect(await screen.findByText("Discard changes?")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Discard" }));
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
  });

  it("keeps the form when the discard prompt is declined", async () => {
    const { onOpenChange } = renderSheet({ dirty: true });
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    await screen.findByText("Discard changes?");
    fireEvent.click(screen.getByRole("button", { name: "Keep as is" }));
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
  });

  it("renders the inline error with role=alert", () => {
    renderSheet({ error: "That name is taken" });
    expect(screen.getByRole("alert")).toHaveTextContent("That name is taken");
  });
});
