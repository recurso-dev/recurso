import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach } from "vitest";
import VerifyEmail from "../VerifyEmail";
import { endpoints } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  endpoints: { verifyEmail: vi.fn() },
}));

// useAuth returns a live holder so a test can swap in a NEW refreshUser identity
// mid-request — the exact condition (AuthProvider re-rendering when its /auth/me
// bootstrap resolves) that used to strand the verify spinner.
const h = vi.hoisted(() => ({
  auth: { refreshUser: null },
}));
vi.mock("@/auth/AuthProvider", () => ({
  useAuth: () => h.auth,
}));

const navigate = vi.fn();
vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal();
  return { ...actual, useNavigate: () => navigate };
});

const routeEl = (route) => (
  <MemoryRouter initialEntries={[route]}>
    <VerifyEmail />
  </MemoryRouter>
);
const renderAt = (route) => render(routeEl(route));

describe("VerifyEmail page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    h.auth = { refreshUser: vi.fn().mockResolvedValue(null) };
  });

  it("verifies the token and shows the success state, refreshing auth", async () => {
    endpoints.verifyEmail.mockResolvedValue({ data: {} });
    renderAt("/verify-email?token=tok_abc");

    await waitFor(() =>
      expect(endpoints.verifyEmail).toHaveBeenCalledWith("tok_abc")
    );
    expect(await screen.findByText(/your email is verified/i)).toBeInTheDocument();
    expect(h.auth.refreshUser).toHaveBeenCalled();
  });

  it("shows the invalid state without calling the API when the token is missing", () => {
    renderAt("/verify-email");
    expect(screen.getByText(/invalid or has expired/i)).toBeInTheDocument();
    expect(endpoints.verifyEmail).not.toHaveBeenCalled();
  });

  it("shows the invalid state when the token is rejected", async () => {
    endpoints.verifyEmail.mockRejectedValue({ response: { status: 400 } });
    renderAt("/verify-email?token=bad");
    expect(await screen.findByText(/invalid or has expired/i)).toBeInTheDocument();
  });

  // Regression: AuthProvider re-renders (giving useAuth a new refreshUser
  // identity) while the verify request is still in flight. Previously the effect
  // depended on refreshUser, so this re-ran it, its cleanup flipped `active` to
  // false, and the resolved request bailed before setStatus — spinner forever.
  it("still reaches success when auth re-renders mid-request (fresh refreshUser)", async () => {
    let resolveVerify;
    endpoints.verifyEmail.mockReturnValue(
      new Promise((r) => {
        resolveVerify = r;
      })
    );

    const { rerender } = renderAt("/verify-email?token=tok_race");
    await waitFor(() =>
      expect(endpoints.verifyEmail).toHaveBeenCalledWith("tok_race")
    );

    // Simulate AuthProvider's bootstrap resolving: new refreshUser identity,
    // then a re-render of the page — all while the POST is still pending.
    h.auth = { refreshUser: vi.fn().mockResolvedValue(null) };
    rerender(routeEl("/verify-email?token=tok_race"));

    // Now the in-flight verification resolves.
    resolveVerify({ data: {} });

    expect(
      await screen.findByText(/your email is verified/i)
    ).toBeInTheDocument();
    // And the single-use token was requested exactly once.
    expect(endpoints.verifyEmail).toHaveBeenCalledTimes(1);
  });
});
