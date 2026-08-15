import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect } from "vitest";

import { DataTable } from "../DataTable";

// Batch D — Table Finish contracts: accessible name, sticky header, header
// semantics. Behavioural/semantic assertions only — no CSS snapshotting.

const columns = [
  { key: "name", header: "Name", cell: (r) => r.name },
  { key: "amount", header: "Amount", align: "right", cell: (r) => r.amount },
];
const data = [
  { id: "1", name: "Acme", amount: "$99.00" },
  { id: "2", name: "Globex", amount: "$10.00" },
];

const renderTable = (props = {}, heading = null) =>
  render(
    <MemoryRouter>
      {heading}
      <DataTable columns={columns} data={data} {...props} />
    </MemoryRouter>
  );

describe("DataTable — accessible name", () => {
  it("names the table from the page heading via aria-labelledby by default", () => {
    renderTable(
      {},
      <h1 id="page-title">Invoices</h1>
    );
    const table = screen.getByRole("table");
    expect(table).toHaveAttribute("aria-labelledby", "page-title");
    // The computed accessible name resolves from the referenced heading.
    expect(screen.getByRole("table", { name: "Invoices" })).toBe(table);
  });

  it("uses an explicit ariaLabel instead of labelledby when given", () => {
    renderTable({ ariaLabel: "Recent payouts" });
    const table = screen.getByRole("table", { name: "Recent payouts" });
    expect(table).toHaveAttribute("aria-label", "Recent payouts");
    expect(table).not.toHaveAttribute("aria-labelledby");
  });

  it("never labels a table with a generic placeholder", () => {
    renderTable({}, <h1 id="page-title">Customers</h1>);
    expect(screen.queryByRole("table", { name: /^data table$/i })).toBeNull();
  });
});

describe("DataTable — sticky header + scroll container", () => {
  it("bounds the scroll wrapper so the header can stick", () => {
    renderTable();
    const wrapper = screen.getByRole("table").parentElement;
    expect(wrapper.className).toContain("overflow-auto");
    expect(wrapper.className).toContain("max-h-[calc(100vh-15rem)]");
  });

  it("makes every header cell sticky and opaque", () => {
    renderTable();
    const headerCells = screen.getAllByRole("columnheader");
    expect(headerCells.length).toBeGreaterThan(0);
    for (const th of headerCells) {
      expect(th.className).toContain("sticky");
      expect(th.className).toContain("top-0");
      expect(th.className).toContain("bg-muted");
    }
  });
});

describe("DataTable — header semantics preserved", () => {
  it("keeps thead + th scope=col", () => {
    renderTable();
    const table = screen.getByRole("table");
    expect(table.querySelector("thead")).toBeTruthy();
    for (const th of table.querySelectorAll("thead th")) {
      expect(th.getAttribute("scope")).toBe("col");
    }
  });

  it("keeps a real accessible name on the select-all header checkbox", () => {
    renderTable({ selectable: true, selectedIds: new Set(), onSelectionChange: () => {} });
    expect(
      screen.getByRole("checkbox", { name: /select all rows on this page/i })
    ).toBeInTheDocument();
  });
});
