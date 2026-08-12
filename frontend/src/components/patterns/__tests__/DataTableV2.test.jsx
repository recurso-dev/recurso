import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi } from "vitest";

import { DataTable } from "../DataTable";
import { moneyColumn } from "../columns";

const wrap = (ui) => render(<MemoryRouter>{ui}</MemoryRouter>);

const rows = [
  { id: "a", name: "Acme", total: 5000, currency: "USD" },
  { id: "b", name: "Beta", total: 100, currency: "USD" },
  { id: "c", name: "Cactus", total: 90000, currency: "USD" },
];

const columns = [
  { key: "name", header: "Name", sortable: true },
  moneyColumn({
    key: "total",
    header: "Amount",
    amount: (r) => r.total,
    currency: (r) => r.currency,
    sortable: true,
  }),
];

describe("DataTable v2 — sorting", () => {
  it("client-sorts on header click and exposes aria-sort", () => {
    wrap(<DataTable columns={columns} data={rows} />);
    const btn = screen.getByRole("button", { name: /Amount/ });
    const th = btn.closest("th");
    expect(th).toHaveAttribute("aria-sort", "none");

    fireEvent.click(btn); // asc
    expect(th).toHaveAttribute("aria-sort", "ascending");
    let cells = screen.getAllByRole("row").slice(1).map((r) => r.textContent);
    expect(cells[0]).toContain("Beta"); // 100 first

    fireEvent.click(btn); // desc
    expect(th).toHaveAttribute("aria-sort", "descending");
    cells = screen.getAllByRole("row").slice(1).map((r) => r.textContent);
    expect(cells[0]).toContain("Cactus"); // 90000 first

    fireEvent.click(btn); // off
    expect(th).toHaveAttribute("aria-sort", "none");
  });

  it("delegates to onSortChange in controlled (server) mode", () => {
    const onSortChange = vi.fn();
    wrap(<DataTable columns={columns} data={rows} sort={null} onSortChange={onSortChange} />);
    fireEvent.click(screen.getByRole("button", { name: /Name/ }));
    expect(onSortChange).toHaveBeenCalledWith({ key: "name", dir: "asc" });
  });
});

describe("DataTable v2 — row semantics", () => {
  it("puts activation on a real button in the first cell, not role=button on the tr", () => {
    const onRowClick = vi.fn();
    wrap(<DataTable columns={columns} data={rows} onRowClick={onRowClick} />);
    const dataRows = screen.getAllByRole("row").slice(1);
    expect(dataRows[0]).not.toHaveAttribute("role", "button");
    expect(dataRows[0]).not.toHaveAttribute("tabindex");

    const activate = screen.getByRole("button", { name: "Acme" });
    fireEvent.click(activate);
    expect(onRowClick).toHaveBeenCalledTimes(1);
  });

  it("does not double-fire when a nested action button is activated", () => {
    const onRowClick = vi.fn();
    const action = vi.fn();
    const cols = [
      ...columns,
      {
        key: "actions",
        header: "",
        cell: () => (
          <button type="button" onClick={(e) => { e.stopPropagation(); action(); }}>
            Copy link
          </button>
        ),
      },
    ];
    wrap(<DataTable columns={cols} data={rows} onRowClick={onRowClick} />);
    const nested = screen.getAllByRole("button", { name: "Copy link" })[0];
    fireEvent.keyDown(nested, { key: "Enter" });
    fireEvent.click(nested); // keyboard Enter on a button fires click
    expect(action).toHaveBeenCalledTimes(1);
    expect(onRowClick).not.toHaveBeenCalled();
  });

  it("shows a chevron affordance on interactive rows", () => {
    const { container } = wrap(
      <DataTable columns={columns} data={rows} onRowClick={() => {}} />
    );
    expect(container.querySelectorAll("tbody td:last-child svg").length).toBe(3);
  });
});

describe("DataTable v2 — pagination contract", () => {
  it("renders an exact range and disables Next on the true boundary", () => {
    const onPageChange = vi.fn();
    wrap(
      <DataTable
        columns={columns}
        data={rows}
        pagination={{ page: 3, pageSize: 25, total: 63, onPageChange }}
      />
    );
    expect(screen.getByText("51–63 of 63")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Next" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Previous" }));
    expect(onPageChange).toHaveBeenCalledWith(2);
  });

  it("still supports the legacy onPrev/onNext shape", () => {
    const onNext = vi.fn();
    wrap(
      <DataTable
        columns={columns}
        data={rows}
        pagination={{ page: 1, onPrev: vi.fn(), onNext, hasNext: true }}
      />
    );
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(onNext).toHaveBeenCalled();
  });
});

describe("DataTable v2 — column priority and footer", () => {
  it("applies hideBelow classes to header and cells", () => {
    const cols = [
      { key: "name", header: "Name" },
      { key: "total", header: "Amount", hideBelow: "md" },
    ];
    const { container } = wrap(<DataTable columns={cols} data={rows} />);
    expect(container.querySelectorAll("th.hidden.md\\:table-cell").length).toBe(1);
    expect(container.querySelectorAll("td.hidden.md\\:table-cell").length).toBe(3);
  });

  it("renders a totals footer", () => {
    wrap(
      <DataTable
        columns={columns}
        data={rows}
        footer={
          <tr>
            <td>Total</td>
            <td>$951.00</td>
          </tr>
        }
      />
    );
    expect(screen.getByText("$951.00")).toBeInTheDocument();
  });
});
