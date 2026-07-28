import { renderHook, waitFor, act } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { AuthProvider, useAuth } from "../AuthProvider";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    authMe: vi.fn(),
    authLogin: vi.fn(),
    loginMfa: vi.fn(),
    authRegister: vi.fn(),
    authLogout: vi.fn(),
    authDemo: vi.fn(),
  },
}));

// authToken touches localStorage on import; jsdom here needs it stubbed.
const store = {};
vi.stubGlobal("localStorage", {
  getItem: (k) => (k in store ? store[k] : null),
  setItem: (k, v) => {
    store[k] = String(v);
  },
  removeItem: (k) => {
    delete store[k];
  },
  clear: () => {
    for (const k in store) delete store[k];
  },
});

const wrapper = ({ children }) => <AuthProvider>{children}</AuthProvider>;

describe("AuthProvider", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.authLogout.mockResolvedValue({ data: {} });
  });

  it("resolves an authenticated session from /auth/me", async () => {
    endpoints.authMe.mockResolvedValue({ data: { user: { id: "u1", email: "a@b.com" } } });
    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.user).toEqual({ id: "u1", email: "a@b.com" });
    expect(result.current.isAuthenticated).toBe(true);
  });

  it("treats a 401 as no session", async () => {
    endpoints.authMe.mockRejectedValue({ response: { status: 401 } });
    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.user).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
    // A 401 is definitive — no retries.
    expect(endpoints.authMe).toHaveBeenCalledTimes(1);
  });

  it("retries a transient (non-401) failure before giving up", async () => {
    vi.useFakeTimers();
    endpoints.authMe
      .mockRejectedValueOnce({ response: { status: 503 } })
      .mockResolvedValueOnce({ data: { user: { id: "u2" } } });

    const { result } = renderHook(() => useAuth(), { wrapper });
    // Let the backoff timer (750ms) elapse so the retry fires.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(800);
    });
    vi.useRealTimers();

    await waitFor(() => expect(result.current.user).toEqual({ id: "u2" }));
    expect(endpoints.authMe).toHaveBeenCalledTimes(2);
  });

  it("login sets the user", async () => {
    endpoints.authMe.mockRejectedValue({ response: { status: 401 } });
    endpoints.authLogin.mockResolvedValue({ data: { user: { id: "u3" } } });
    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.login("a@b.com", "pw");
    });
    expect(result.current.user).toEqual({ id: "u3" });
    expect(endpoints.authLogin).toHaveBeenCalledWith("a@b.com", "pw");
  });

  it("logout clears the user and calls the API", async () => {
    endpoints.authMe.mockResolvedValue({ data: { user: { id: "u1" } } });
    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.user).not.toBeNull());

    await act(async () => {
      await result.current.logout();
    });
    expect(endpoints.authLogout).toHaveBeenCalled();
    expect(result.current.user).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
  });

  it("stays authenticated in legacy API-key mode", async () => {
    endpoints.authMe.mockRejectedValue({ response: { status: 401 } });
    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));

    act(() => result.current.loginWithApiKey("sk_test_123"));
    expect(result.current.apiKey).toBe("sk_test_123");
    expect(result.current.isAuthenticated).toBe(true);
  });
});
