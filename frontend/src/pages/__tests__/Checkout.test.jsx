import { render, screen, waitFor, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { describe, it, expect, vi, afterEach } from "vitest";

import Checkout from "../Checkout";

// Stripe is loaded lazily and mounts an <Elements> tree; stub both packages so
// the checkout renders in jsdom without a real Stripe.js.
vi.mock("@stripe/stripe-js", () => ({ loadStripe: vi.fn(() => Promise.resolve(null)) }));
vi.mock("@stripe/react-stripe-js", () => ({
    Elements: ({ children }) => <div>{children}</div>,
    PaymentElement: () => <div data-testid="payment-element" />,
    AddressElement: () => <div data-testid="address-element" />,
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
        // The pay CTA and the totals both carry the amount, rendered through the
        // exponent-aware formatter so the currency shows as its symbol ($ for
        // USD) consistently — not a mix of "$42.00" and "USD 42.00".
        await waitFor(() => expect(screen.getAllByText(/\$42\.00/).length).toBeGreaterThan(0));
    });

    it("asks for the amount due, not the total, when credit was applied", async () => {
        global.fetch = vi.fn(() =>
            Promise.resolve({
                ok: true,
                json: () => Promise.resolve({ data: invoice({ total: 5000, amount_due: 3000 }) }),
            })
        );
        renderAt("inv_credit");
        expect(await screen.findByRole("button", { name: /pay \$30\.00/i })).toBeInTheDocument();
        expect(screen.getByText("Amount due")).toBeInTheDocument();
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

    describe("telling the customer their card will be saved", () => {
        const startPayment = async (saves) => {
            global.fetch = vi.fn((url, opts) => {
                const data =
                    opts?.method === "POST"
                        ? {
                              gateway: "stripe",
                              client_secret: "secret",
                              publishable_key: "pk_test",
                              saves_payment_method: saves,
                          }
                        : invoice();
                return Promise.resolve({ ok: true, json: () => Promise.resolve({ data }) });
            });
            renderAt("inv_1");
            fireEvent.click(await screen.findByRole("button", { name: /pay/i }));
            await screen.findByTestId("payment-element");
        };

        it("discloses saving when the order will save the card", async () => {
            await startPayment(true);
            expect(screen.getByText(/saved and charged automatically/i)).toBeInTheDocument();
        });

        it("says nothing when the order won't save the card", async () => {
            await startPayment(false);
            expect(screen.queryByText(/saved and charged automatically/i)).not.toBeInTheDocument();
        });
    });
});
