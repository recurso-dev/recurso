import { describe, it, expect, vi } from "vitest";

// The request interceptor injects the in-memory API key. It must NOT override
// an explicit per-request Authorization header: verifyApiKey probes a freshly
// pasted key while an older key may still be held in memory, and the old
// behaviour verified the held key instead of the pasted one.
const inst = {
  get: vi.fn(() => Promise.resolve({ data: {} })),
  post: vi.fn(() => Promise.resolve({ data: {} })),
  put: vi.fn(() => Promise.resolve({ data: {} })),
  patch: vi.fn(() => Promise.resolve({ data: {} })),
  delete: vi.fn(() => Promise.resolve({ data: {} })),
  interceptors: { request: { use: vi.fn() } },
  defaults: {},
};
vi.mock("axios", () => ({
  default: { create: () => inst, get: vi.fn(), post: vi.fn(), defaults: {} },
}));
vi.mock("../authToken", () => ({ getApiKey: () => "held_key" }));

await import("../api");
const onRequest = inst.interceptors.request.use.mock.calls[0][0];

describe("api request interceptor", () => {
  it("injects the held key when the request carries no Authorization header", () => {
    const config = onRequest({ headers: {} });
    expect(config.headers.Authorization).toBe("Bearer held_key");
  });

  it("leaves an explicit Authorization header alone", () => {
    const config = onRequest({ headers: { Authorization: "Bearer pasted_key" } });
    expect(config.headers.Authorization).toBe("Bearer pasted_key");
  });
});
