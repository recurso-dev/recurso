import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import PortalPaymentMethod from "../PortalPaymentMethod";

// Stripe never runs in jsdom: loadStripe resolves to a fake client whose
// confirmSetup outcome each test controls, and the Elements wrappers are inert.
const stripeClient = { confirmSetup: vi.fn() };
vi.mock("@stripe/stripe-js", () => ({ loadStripe: vi.fn(() => Promise.resolve(stripeClient)) }));
vi.mock("@stripe/react-stripe-js", () => ({
  Elements: ({ children }) => <div data-testid="elements">{children}</div>,
  PaymentElement: () => <div data-testid="payment-element" />,
  AddressElement: () => <div data-testid="address-element" />,
  useStripe: () => stripeClient,
  useElements: () => ({}),
}));

const AUTH = { Authorization: "Bearer portal_tok" };
const API = "https://api.test";

const jsonResponse = (status, body) =>
  Promise.resolve({ ok: status >= 200 && status < 300, status, json: () => Promise.resolve(body) });

const renderDialog = (props = {}) =>
  render(
    <PortalPaymentMethod open onOpenChange={() => {}} apiBase={API} authHeaders={AUTH} {...props} />
  );

beforeEach(() => {
  vi.clearAllMocks();
  global.fetch = vi.fn();
});
afterEach(() => vi.restoreAllMocks());

describe("PortalPaymentMethod", () => {
  it("requests a card SetupIntent with the portal auth headers on open", async () => {
    global.fetch.mockImplementation(() =>
      jsonResponse(200, { data: { client_secret: "seti_secret", publishable_key: "pk_test" } })
    );
    renderDialog();
    expect(screen.getByText("Update payment method")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId("payment-element")).toBeInTheDocument());
    const [url, opts] = global.fetch.mock.calls[0];
    expect(url).toBe(`${API}/portal/api/payment-method/setup-intent`);
    expect(opts).toMatchObject({ method: "POST", credentials: "include", headers: AUTH });
  });

  it("switches to the ACH SetupIntent when the bank method is chosen", async () => {
    global.fetch.mockImplementation(() =>
      jsonResponse(200, { data: { client_secret: "seti_bank", publishable_key: "pk_test" } })
    );
    renderDialog();
    fireEvent.click(screen.getByRole("button", { name: /us bank \(ach\)/i }));
    await waitFor(() =>
      expect(global.fetch).toHaveBeenCalledWith(
        `${API}/portal/api/payment-method/bank-setup-intent`,
        expect.objectContaining({ method: "POST" })
      )
    );
    expect(await screen.findByLabelText("Account holder name")).toBeInTheDocument();
    // The bank flow refuses to start without holder details.
    fireEvent.click(screen.getByRole("button", { name: /connect bank account/i }));
    expect(await screen.findByText(/Enter the account holder's name and email/)).toBeInTheDocument();
  });

  it("confirms the card with Stripe, then finalizes server-side and shows success", async () => {
    const onSaved = vi.fn();
    global.fetch.mockImplementation((url) => {
      if (String(url).endsWith("/setup-intent")) {
        return jsonResponse(200, { data: { client_secret: "seti_secret", publishable_key: "pk_test" } });
      }
      if (String(url).endsWith("/confirm")) {
        return jsonResponse(200, { data: { status: "saved", card: { brand: "visa", last4: "4242" } } });
      }
      return jsonResponse(404, {});
    });
    stripeClient.confirmSetup.mockResolvedValue({ setupIntent: { id: "seti_1", status: "succeeded" } });
    renderDialog({ onSaved });
    await waitFor(() => expect(screen.getByTestId("payment-element")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /save card/i }));

    await waitFor(() => expect(screen.getByText("Your card has been updated.")).toBeInTheDocument());
    const confirmCall = global.fetch.mock.calls.find(([u]) => String(u).endsWith("/confirm"));
    expect(confirmCall[1]).toMatchObject({ method: "POST", credentials: "include" });
    expect(JSON.parse(confirmCall[1].body)).toEqual({ setup_intent_id: "seti_1" });
    expect(onSaved).toHaveBeenCalledWith({ brand: "visa", last4: "4242" });
  });

  it("shows Stripe's decline message and does not finalize", async () => {
    global.fetch.mockImplementation(() =>
      jsonResponse(200, { data: { client_secret: "seti_secret", publishable_key: "pk_test" } })
    );
    stripeClient.confirmSetup.mockResolvedValue({ error: { message: "Your card was declined." } });
    renderDialog();
    await waitFor(() => expect(screen.getByTestId("payment-element")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /save card/i }));
    expect(await screen.findByText("Your card was declined.")).toBeInTheDocument();
    expect(global.fetch.mock.calls.some(([u]) => String(u).endsWith("/confirm"))).toBe(false);
  });

  it("falls back to UPI mandate re-authorization when self-serve setup is unavailable", async () => {
    global.fetch.mockImplementation((url) => {
      if (String(url).endsWith("/mandate")) {
        return jsonResponse(400, { error: { message: "no active mandate" } });
      }
      return jsonResponse(503, {});
    });
    renderDialog();
    expect(await screen.findByText(/Self-serve setup isn't available/)).toBeInTheDocument();
    // Method selector is hidden on the fallback path.
    expect(screen.queryByRole("button", { name: /us bank \(ach\)/i })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: /re-authorize upi autopay/i }));
    await waitFor(() =>
      expect(global.fetch).toHaveBeenCalledWith(
        `${API}/portal/api/payment-method/mandate`,
        expect.objectContaining({ method: "POST", headers: AUTH })
      )
    );
    expect(await screen.findByText("no active mandate")).toBeInTheDocument();
  });

  it("surfaces the server error when the SetupIntent cannot be created", async () => {
    global.fetch.mockImplementation(() =>
      jsonResponse(500, { error: { message: "gateway unavailable" } })
    );
    renderDialog();
    expect(await screen.findByText("gateway unavailable")).toBeInTheDocument();
  });
});
