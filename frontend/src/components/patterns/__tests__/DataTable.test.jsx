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

describe("DataTable new-row reveal", () => {
  const cols = [{ key: "name", header: "Name", cell: (r) => r.name }];
  const trOf = (text) => screen.getByText(text).closest("tr");

  it("does not animate rows on first mount", () => {
    render(
      <MemoryRouter>
        <DataTable columns={cols} data={[{ id: "1", name: "A" }]} />
      </MemoryRouter>
    );
    expect(trOf("A")).not.toHaveClass("animate-motion-reveal");
  });

  it("animates a newly added row but not the rows that persist", () => {
    const { rerender } = render(
      <MemoryRouter>
        <DataTable columns={cols} data={[{ id: "1", name: "A" }]} />
      </MemoryRouter>
    );
    rerender(
      <MemoryRouter>
        <DataTable
          columns={cols}
          data={[{ id: "1", name: "A" }, { id: "2", name: "B" }]}
        />
      </MemoryRouter>
    );
    expect(trOf("B")).toHaveClass("animate-motion-reveal");
    expect(trOf("A")).not.toHaveClass("animate-motion-reveal");
  });
});
