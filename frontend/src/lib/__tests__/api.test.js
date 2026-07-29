import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock axios so `api = axios.create()` returns a spyable instance and the
// endpoint helpers just record the (method, url, body) they would send.
const inst = {
  get: vi.fn(() => Promise.resolve({ data: {} })),
  post: vi.fn(() => Promise.resolve({ data: {} })),
  put: vi.fn(() => Promise.resolve({ data: {} })),
  patch: vi.fn(() => Promise.resolve({ data: {} })),
  delete: vi.fn(() => Promise.resolve({ data: {} })),
  interceptors: { request: { use: vi.fn() } },
  defaults: {},
};
vi.mock("axios", () => {
  const axios = {
    create: () => inst,
    get: vi.fn(() => Promise.resolve({ data: {} })),
    post: vi.fn(() => Promise.resolve({ data: {} })),
    defaults: {},
  };
  return { default: axios };
});
vi.mock("../authToken", () => ({ getApiKey: () => "" }));

const { endpoints } = await import("../api");

const lastCall = (spy) => spy.mock.calls.at(-1);

describe("api endpoint contracts", () => {
  beforeEach(() => {
    inst.get.mockClear();
    inst.post.mockClear();
    inst.put.mockClear();
    inst.delete.mockClear();
    inst.patch.mockClear();
  });

  it("builds tenant-scoped resource URLs correctly", () => {
    endpoints.getCustomer("cus_1");
    expect(lastCall(inst.get)[0]).toBe("/customers/cus_1");

    endpoints.getInvoicePdf("inv_1");
    expect(lastCall(inst.get)[0]).toBe("/invoices/inv_1/pdf");
    expect(lastCall(inst.get)[1]).toMatchObject({ responseType: "blob" });
  });

  it("maps money-path mutations to the right verb + path", () => {
    endpoints.voidCreditNote("cn_1");
    expect(lastCall(inst.post)[0]).toBe("/credit-notes/cn_1/void");

    endpoints.getCreditNotePdf("cn_1");
    expect(lastCall(inst.get)[0]).toBe("/credit-notes/cn_1/pdf");

    endpoints.cancelSubscription("sub_1", { reason: "too_expensive" });
    expect(lastCall(inst.post)[0]).toBe("/subscriptions/sub_1/cancel");
    expect(lastCall(inst.post)[1]).toEqual({ reason: "too_expensive" });

    endpoints.resolveDispute("d_1", { outcome: "accept" });
    expect(lastCall(inst.post)[0]).toBe("/disputes/d_1/resolve");
  });

  it("maps lifecycle actions correctly", () => {
    endpoints.approveCreditNote("cn_1");
    expect(lastCall(inst.post)[0]).toBe("/credit-notes/cn_1/approve");
    endpoints.deleteQuote("q_1");
    expect(lastCall(inst.delete)[0]).toBe("/quotes/q_1");
    endpoints.setCouponActive("cp_1", false);
    expect(lastCall(inst.put)[0]).toBe("/coupons/cp_1");
    expect(lastCall(inst.put)[1]).toEqual({ active: false });
    endpoints.revokeMandate("m_1");
    expect(lastCall(inst.post)[0]).toBe("/mandates/m_1/revoke");
  });

  it("passes query params through on list endpoints", () => {
    endpoints.getSubscriptions({ limit: 25, status: "active" });
    expect(lastCall(inst.get)[0]).toBe("/subscriptions");
    expect(lastCall(inst.get)[1]).toMatchObject({ params: { limit: 25, status: "active" } });
  });
});
