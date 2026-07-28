import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect } from "vitest";
import { DataTable } from "../DataTable";

const columns = [{ key: "name", header: "Name", cell: (r) => r.name }];

const renderAt = (route, props = {}) =>
  render(
    <MemoryRouter initialEntries={[route]}>
      <DataTable columns={columns} data={[]} {...props} />
    </MemoryRouter>
  );

describe("DataTable empty-state docs link", () => {
  it("shows a contextual guide link on the getting-started empty state", () => {
    renderAt("/customers", { empty: { title: "No customers yet" } });
    const link = screen.getByText("Read the guide").closest("a");
    expect(link).toHaveAttribute("href", "https://docs.recurso.dev/dashboard/customers");
  });

  it("hides the guide link while a search is active", () => {
    renderAt("/customers", {
      search: { value: "acme", onChange: () => {} },
      empty: { title: "No matching customers" },
    });
    expect(screen.queryByText("Read the guide")).not.toBeInTheDocument();
  });

  it("can be suppressed with docsLink={false}", () => {
    renderAt("/customers", { docsLink: false, empty: { title: "No customers yet" } });
    expect(screen.queryByText("Read the guide")).not.toBeInTheDocument();
  });
});
