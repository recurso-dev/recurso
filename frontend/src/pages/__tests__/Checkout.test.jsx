import { render, screen, waitFor, cleanup } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { describe, it, expect, vi, afterEach } from "vitest";

import Checkout from "../Checkout";

// Stripe is loaded lazily and mounts an <Elements> tree; stub both packages so
// the checkout renders in jsdom without a real Stripe.js.
vi.mock("@stripe/stripe-js", () => ({ loadStripe: vi.fn(() => Promise.resolve(null)) }));
vi.mock("@stripe/react-stripe-js", () => ({
    Elements: ({ children }) => <div>{children}</div>,
    PaymentElement: () => <div data-testid="payment-element" />,
    useStripe: () => null,
    useElements: () => null,
}));

const invoice = (over = {}) => ({
    status: "open",
    currency: "USD",
    display_amount: "42.00",
    subtotal: 4000,
    total: 4200,
    ...over,
});

const renderAt = (id) =>
    render(
        <MemoryRouter initialEntries={[`/checkout/${id}`]}>
            <Routes>
                <Route path="/checkout/:id" element={<Checkout />} />
            </Routes>
        </MemoryRouter>
    );

describe("Checkout (money-critical public flow)", () => {
    afterEach(() => {
        cleanup();
        vi.restoreAllMocks();
    });

    it("loads the invoice and shows the amount to pay", async () => {
        global.fetch = vi.fn(() =>
            Promise.resolve({ ok: true, json: () => Promise.resolve({ data: invoice() }) })
        );
        renderAt("inv_1");
        // The pay CTA and the totals both carry the amount.
        await waitFor(() => expect(screen.getAllByText(/42\.00/).length).toBeGreaterThan(0));
        expect(screen.getAllByText(/USD/).length).toBeGreaterThan(0);
    });

    it("shows a not-found error when the invoice can't be loaded", async () => {
        global.fetch = vi.fn(() => Promise.resolve({ ok: false }));
        renderAt("missing");
        await waitFor(() => expect(screen.getAllByText(/not found/i).length).toBeGreaterThan(0));
    });

    it("reflects an already-paid invoice", async () => {
        global.fetch = vi.fn(() =>
            Promise.resolve({ ok: true, json: () => Promise.resolve({ data: invoice({ status: "paid" }) }) })
        );
        renderAt("inv_paid");
        await waitFor(() => expect(screen.getByText(/paid/i)).toBeInTheDocument());
    });
});
