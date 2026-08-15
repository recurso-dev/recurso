import { useState } from "react";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route, useLocation } from "react-router";
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

describe("DataTable selection", () => {
  const cols = [{ key: "name", header: "Name", cell: (r) => r.name }];
  const three = [
    { id: "a", name: "A" },
    { id: "b", name: "B" },
    { id: "c", name: "C" },
  ];

  function Harness({ data = three, rowHref }) {
    const [ids, setIds] = useState(new Set());
    return (
      <DataTable
        columns={cols}
        data={data}
        rowHref={rowHref}
        selectable
        selectedIds={ids}
        onSelectionChange={setIds}
        renderBulkActions={(sel) => <span>bulk:{sel.size}</span>}
      />
    );
  }

  const renderH = (props) =>
    render(
      <MemoryRouter>
        <Harness {...props} />
      </MemoryRouter>
    );

  const rowCheckbox = (name) =>
    screen.getByRole("checkbox", { name: `Select row ${name}` });
  const headerCheckbox = () =>
    screen.getByRole("checkbox", { name: "Select all rows on this page" });

  it("shows no bulk bar when nothing is selected", () => {
    renderH();
    expect(screen.queryByText(/selected/)).toBeNull();
  });

  it("selects an individual row", () => {
    renderH();
    fireEvent.click(rowCheckbox("a"));
    expect(screen.getByText("1 selected")).toBeInTheDocument();
    expect(screen.getByText("bulk:1")).toBeInTheDocument();
    expect(rowCheckbox("a")).toBeChecked();
    expect(rowCheckbox("b")).not.toBeChecked();
  });

  it("select-all selects every row on this page, and toggles off", () => {
    renderH();
    fireEvent.click(headerCheckbox());
    expect(screen.getByText("3 selected")).toBeInTheDocument();
    expect(rowCheckbox("a")).toBeChecked();
    expect(rowCheckbox("c")).toBeChecked();
    // The label makes the page scope unmistakable.
    expect(screen.getByText("on this page")).toBeInTheDocument();
    fireEvent.click(headerCheckbox());
    expect(screen.queryByText(/selected/)).toBeNull();
  });

  it("deselects a single row", () => {
    renderH();
    fireEvent.click(headerCheckbox());
    fireEvent.click(rowCheckbox("b"));
    expect(screen.getByText("2 selected")).toBeInTheDocument();
    expect(rowCheckbox("b")).not.toBeChecked();
  });

  it("Clear empties the selection", () => {
    renderH();
    fireEvent.click(headerCheckbox());
    fireEvent.click(screen.getByRole("button", { name: "Clear" }));
    expect(screen.queryByText(/selected/)).toBeNull();
  });

  it("retains valid selections when the same page refetches", async () => {
    const { rerender } = renderH();
    fireEvent.click(headerCheckbox());
    expect(screen.getByText("3 selected")).toBeInTheDocument();
    // Refetch: a new array with the SAME ids.
    rerender(
      <MemoryRouter>
        <Harness data={three.map((r) => ({ ...r }))} />
      </MemoryRouter>
    );
    // Selection survives (ids still present).
    expect(screen.getByText("3 selected")).toBeInTheDocument();
  });

  it("prunes selections that leave the result set (filter/paginate)", async () => {
    const { rerender } = renderH();
    fireEvent.click(headerCheckbox());
    expect(screen.getByText("3 selected")).toBeInTheDocument();
    // Paginate/filter to a disjoint set — the old ids are no longer valid.
    rerender(
      <MemoryRouter>
        <Harness data={[{ id: "x", name: "X" }]} />
      </MemoryRouter>
    );
    await waitFor(() => expect(screen.queryByText(/selected/)).toBeNull());
    expect(screen.getByRole("checkbox", { name: "Select row x" })).not.toBeChecked();
  });

  it("selecting a row does not navigate (checkbox stops propagation)", () => {
    function LocationProbe() {
      const loc = useLocation();
      return <div data-testid="loc">{loc.pathname}</div>;
    }
    render(
      <MemoryRouter initialEntries={["/list"]}>
        <Routes>
          <Route
            path="/list"
            element={
              <>
                <Harness rowHref={(r) => `/x/${r.id}`} />
                <LocationProbe />
              </>
            }
          />
          <Route path="/x/:id" element={<div>detail</div>} />
        </Routes>
      </MemoryRouter>
    );
    fireEvent.click(rowCheckbox("a"));
    expect(screen.getByTestId("loc")).toHaveTextContent("/list");
    expect(screen.queryByText("detail")).toBeNull();
    expect(rowCheckbox("a")).toBeChecked();
  });

  it("checkboxes are keyboard-operable", async () => {
    const user = userEvent.setup();
    renderH();
    await user.tab(); // focus the header checkbox (first focusable)
    expect(headerCheckbox()).toHaveFocus();
    await user.keyboard(" ");
    expect(screen.getByText("3 selected")).toBeInTheDocument();
  });
});
