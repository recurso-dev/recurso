import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi } from "vitest";
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

describe("DataTable pagination contract", () => {
  const cols = [{ key: "name", header: "Name", cell: (r) => r.name }];
  const rows = (n) => Array.from({ length: n }, (_, i) => ({ id: String(i), name: `R${i}` }));

  const renderPaged = (pagination, data = rows(2)) =>
    render(
      <MemoryRouter>
        <DataTable columns={cols} data={data} pagination={pagination} />
      </MemoryRouter>
    );

  it("shows the start–end of total range and page count with the total contract", () => {
    // Middle page of 130 rows at 25/page → 26–50 of 130, page 2 / 6.
    renderPaged({ page: 2, pageSize: 25, total: 130, onPageChange: () => {} });
    expect(screen.getByText("26–50 of 130")).toBeInTheDocument();
    expect(screen.getByText("2 / 6")).toBeInTheDocument();
  });

  it("disables Previous on the first page and Next on the last", () => {
    const { rerender } = renderPaged({ page: 1, pageSize: 25, total: 130, onPageChange: () => {} });
    expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Next" })).toBeEnabled();

    rerender(
      <MemoryRouter>
        <DataTable columns={cols} data={rows(2)} pagination={{ page: 6, pageSize: 25, total: 130, onPageChange: () => {} }} />
      </MemoryRouter>
    );
    expect(screen.getByRole("button", { name: "Previous" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Next" })).toBeDisabled();
  });

  it("calls onPageChange with the neighbouring page", () => {
    const onPageChange = vi.fn();
    renderPaged({ page: 2, pageSize: 25, total: 130, onPageChange });
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(onPageChange).toHaveBeenCalledWith(3);
    fireEvent.click(screen.getByRole("button", { name: "Previous" }));
    expect(onPageChange).toHaveBeenCalledWith(1);
  });

  it("supports the legacy hasNext contract (no total)", () => {
    const onNext = vi.fn();
    renderPaged({ page: 2, onPrev: () => {}, onNext, hasNext: true });
    const next = screen.getByRole("button", { name: "Next" });
    expect(next).toBeEnabled();
    fireEvent.click(next);
    expect(onNext).toHaveBeenCalled();
  });

  it("hides pagination while loading (no controls to jump an empty page)", () => {
    render(
      <MemoryRouter>
        <DataTable columns={cols} data={[]} loading pagination={{ page: 1, pageSize: 25, total: 130, onPageChange: () => {} }} />
      </MemoryRouter>
    );
    expect(screen.queryByRole("button", { name: "Next" })).toBeNull();
  });
});
