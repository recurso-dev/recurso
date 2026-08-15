import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router";
import { describe, it, expect } from "vitest";

import { ObjectHeader } from "../ObjectPage";
import { DataTable } from "../DataTable";

// Batch F1 — context-preserving back navigation. A DataTable row stashes its
// originating list URL as navigation state; the object page's back-link returns
// there exactly. Direct opens / cross-object navigation fall back to the list root.

const renderHeaderAt = (state) =>
  render(
    <MemoryRouter initialEntries={[{ pathname: "/invoices/1", state }]}>
      <ObjectHeader backTo="/invoices" backLabel="Invoices" kicker="Invoice" title="INV-1" />
    </MemoryRouter>
  );

const backLink = () => screen.getByRole("link", { name: /Invoices/ });

describe("ObjectHeader back-link resolution", () => {
  it("restores the originating filtered list when state.from matches backTo", () => {
    renderHeaderAt({ from: "/invoices?status=past_due&page=2" });
    expect(backLink()).toHaveAttribute("href", "/invoices?status=past_due&page=2");
  });

  it("restores multi-filter + search + sort query intact", () => {
    renderHeaderAt({ from: "/invoices?q=acme&status=open&sort=amount&dir=desc&page=3" });
    expect(backLink()).toHaveAttribute(
      "href",
      "/invoices?q=acme&status=open&sort=amount&dir=desc&page=3"
    );
  });

  it("falls back to the list root on a direct object open (no navigation state)", () => {
    renderHeaderAt(undefined);
    expect(backLink()).toHaveAttribute("href", "/invoices");
  });

  it("falls back to the list root when from is a different list (cross-object nav)", () => {
    renderHeaderAt({ from: "/customers?q=acme" });
    expect(backLink()).toHaveAttribute("href", "/invoices");
  });

  it("does not treat a sibling sub-path as the owning list", () => {
    // "/payments/offline" must not be mistaken for the "/payments" list.
    render(
      <MemoryRouter initialEntries={[{ pathname: "/payments/1", state: { from: "/payments/offline?x=1" } }]}>
        <ObjectHeader backTo="/payments" backLabel="Payments" title="P-1" />
      </MemoryRouter>
    );
    expect(screen.getByRole("link", { name: /Payments/ })).toHaveAttribute("href", "/payments");
  });
});

describe("end-to-end: filtered list → row → object back-link", () => {
  const columns = [{ key: "name", header: "Name", cell: (r) => r.name }];
  const data = [{ id: "1", name: "Acme" }];

  const App = () => (
    <Routes>
      <Route
        path="/invoices"
        element={<DataTable columns={columns} data={data} rowHref={(r) => `/invoices/${r.id}`} />}
      />
      <Route
        path="/invoices/:id"
        element={<ObjectHeader backTo="/invoices" backLabel="Invoices" title="INV-1" />}
      />
    </Routes>
  );

  it("carries the list filters through the row link to the object's back-link", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={["/invoices?status=past_due&page=2"]}>
        <App />
      </MemoryRouter>
    );
    await user.click(screen.getByRole("link", { name: "Acme" }));
    // Now on the object page; the back-link must restore the exact filtered list.
    expect(screen.getByRole("link", { name: /Invoices/ })).toHaveAttribute(
      "href",
      "/invoices?status=past_due&page=2"
    );
  });
});
