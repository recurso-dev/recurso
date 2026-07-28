import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { EmptyState } from "../EmptyState";

describe("EmptyState", () => {
  it("renders title and description", () => {
    render(<EmptyState title="Nothing yet" description="Add your first record." />);
    expect(screen.getByText("Nothing yet")).toBeInTheDocument();
    expect(screen.getByText("Add your first record.")).toBeInTheDocument();
  });

  it("renders a learn-more link when learnMoreHref is provided", () => {
    render(
      <EmptyState
        title="Single-entity account"
        learnMoreHref="https://docs.recurso.dev/dashboard/entities"
        learnMoreLabel="Set up multiple entities"
      />
    );
    const link = screen.getByText("Set up multiple entities").closest("a");
    expect(link).toHaveAttribute("href", "https://docs.recurso.dev/dashboard/entities");
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("omits the learn-more link by default", () => {
    render(<EmptyState title="Empty" />);
    expect(screen.queryByText("Read the guide")).not.toBeInTheDocument();
  });
});
