import { render, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

const navigateMock = vi.fn();
vi.mock("react-router", async () => {
  const actual = await vi.importActual("react-router");
  return { ...actual, useNavigate: () => navigateMock };
});

import PortalVerify from "../PortalVerify";

beforeEach(() => {
  global.fetch = vi.fn();
  navigateMock.mockClear();
  window.history.replaceState({}, "", "/portal/verify?token=magic-123");
});
afterEach(() => vi.restoreAllMocks());

const renderAt = () =>
  render(
    <MemoryRouter initialEntries={["/portal/verify?token=magic-123"]}>
      <PortalVerify />
    </MemoryRouter>
  );

describe("PortalVerify (#490 hardening locked)", () => {
  it("POSTs the token in the body (not the query) and navigates to the dashboard on success", async () => {
    global.fetch.mockResolvedValue({ ok: true });
    renderAt();

    await waitFor(() => expect(global.fetch).toHaveBeenCalled());
    const [url, opts] = global.fetch.mock.calls[0];
    // Token must NOT be in the URL, and must be POSTed in the body.
    expect(url).toMatch(/\/portal\/auth\/verify$/);
    expect(url).not.toContain("token=");
    expect(opts.method).toBe("POST");
    expect(JSON.parse(opts.body)).toEqual({ token: "magic-123" });

    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/portal/dashboard"));
  });

  it("navigates to login with an error when verification fails", async () => {
    global.fetch.mockResolvedValue({ ok: false });
    renderAt();
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/portal/login?error=invalid"));
  });
});
