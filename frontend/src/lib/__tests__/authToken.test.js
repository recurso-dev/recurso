import { describe, it, expect, beforeEach, vi } from "vitest";

// The module runs a one-time localStorage cleanup on import, and must never
// write the key to storage — stub localStorage so we can assert both.
const store = {};
const setItem = vi.fn((k, v) => {
  store[k] = String(v);
});
const removeItem = vi.fn((k) => {
  delete store[k];
});
vi.stubGlobal("localStorage", {
  getItem: (k) => (k in store ? store[k] : null),
  setItem,
  removeItem,
  clear: () => {
    for (const k in store) delete store[k];
  },
});
vi.stubGlobal("window", { localStorage });

import { getApiKey, setApiKey, clearApiKey } from "../authToken";

describe("authToken", () => {
  beforeEach(() => {
    clearApiKey();
    setItem.mockClear();
  });

  it("stores and returns the key in memory", () => {
    expect(getApiKey()).toBe("");
    setApiKey("sk_test_abc");
    expect(getApiKey()).toBe("sk_test_abc");
  });

  it("clears the key", () => {
    setApiKey("sk_test_abc");
    clearApiKey();
    expect(getApiKey()).toBe("");
  });

  it("treats null/undefined as an empty key", () => {
    setApiKey(null);
    expect(getApiKey()).toBe("");
    setApiKey(undefined);
    expect(getApiKey()).toBe("");
  });

  it("NEVER persists the key to localStorage (XSS hardening)", () => {
    setApiKey("sk_test_secret");
    expect(setItem).not.toHaveBeenCalled();
    expect(localStorage.getItem("recurso_api_key")).toBeNull();
  });
});
