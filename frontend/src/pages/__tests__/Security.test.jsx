import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Security from "../Security";
import { endpoints } from "../../lib/api";
import { toast } from "@/components/ui/sonner";

vi.mock("../../lib/api", () => ({
  endpoints: {
    mfaSetup: vi.fn(),
    mfaVerify: vi.fn(),
    mfaDisable: vi.fn(),
    getSessions: vi.fn(),
    revokeSession: vi.fn(),
    revokeOtherSessions: vi.fn(),
    getSSOConnection: vi.fn(),
    updateSSOConnection: vi.fn(),
    deleteSSOConnection: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
// QR rendering is Stripe-of-the-moment irrelevant here; keep the secret visible.
vi.mock("qrcode.react", () => ({ QRCodeSVG: () => <svg data-testid="qr" /> }));

let authUser = { id: "u1", role: "owner", mfa_enabled: false };
vi.mock("@/auth/AuthProvider", () => ({ useAuth: () => ({ user: authUser }) }));

const wrapper = ({ children }) => (
  <QueryClientProvider
    client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
  >
    {children}
  </QueryClientProvider>
);

const SESSIONS = [
  { id: "s1", current: true, user_agent: "Mozilla/5.0 (Macintosh; Mac OS X) Chrome/120 Safari/537", created_at: "2026-08-01T00:00:00Z", expires_at: "2026-09-01T00:00:00Z" },
  { id: "s2", current: false, user_agent: "Mozilla/5.0 (Windows NT 10.0) Firefox/128", created_at: "2026-08-02T00:00:00Z", expires_at: "2026-09-02T00:00:00Z" },
];

describe("Security page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authUser = { id: "u1", role: "owner", mfa_enabled: false };
    endpoints.getSessions.mockResolvedValue({ data: { data: SESSIONS } });
    endpoints.getSSOConnection.mockRejectedValue({ response: { status: 404 } });
    endpoints.revokeSession.mockResolvedValue({ data: {} });
    endpoints.revokeOtherSessions.mockResolvedValue({ data: {} });
  });

  it("lists active sessions as readable devices and revokes another session", async () => {
    render(<Security />, { wrapper });
    expect(screen.getByText("Security")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText("Chrome · macOS")).toBeInTheDocument());
    expect(screen.getByText("Firefox · Windows")).toBeInTheDocument();
    expect(screen.getByText("This device")).toBeInTheDocument();
    // Only the non-current session can be revoked individually.
    const revoke = screen.getAllByRole("button", { name: /^revoke$/i });
    expect(revoke).toHaveLength(1);
    fireEvent.click(revoke[0]);
    await waitFor(() => expect(endpoints.revokeSession).toHaveBeenCalledWith("s2"));

    fireEvent.click(screen.getByRole("button", { name: /log out all other sessions/i }));
    await waitFor(() => expect(endpoints.revokeOtherSessions).toHaveBeenCalled());
  });

  it("walks through MFA setup: secret → code → one-time backup codes", async () => {
    endpoints.mfaSetup.mockResolvedValue({
      data: { secret: "JBSWY3DPEHPK3PXP", otpauth_url: "otpauth://totp/Recurso?secret=JBSWY3DPEHPK3PXP" },
    });
    endpoints.mfaVerify.mockResolvedValue({ data: { backup_codes: ["aaaa-1111", "bbbb-2222"] } });
    render(<Security />, { wrapper });

    fireEvent.click(screen.getByRole("button", { name: /enable two-factor authentication/i }));
    expect(await screen.findByText("JBSWY3DPEHPK3PXP")).toBeInTheDocument();
    expect(screen.getByTestId("qr")).toBeInTheDocument();

    const verify = screen.getByRole("button", { name: /verify & enable/i });
    expect(verify).toBeDisabled(); // needs a 6-digit code
    fireEvent.change(document.getElementById("mfa-verify"), { target: { value: "12a456" } });
    // Non-digits are stripped, so this is still too short.
    expect(verify).toBeDisabled();
    fireEvent.change(document.getElementById("mfa-verify"), { target: { value: "123456" } });
    fireEvent.click(verify);

    await waitFor(() => expect(endpoints.mfaVerify).toHaveBeenCalledWith("123456"));
    expect(await screen.findByText("aaaa-1111")).toBeInTheDocument();
    expect(screen.getByText("bbbb-2222")).toBeInTheDocument();
    expect(toast.success).toHaveBeenCalledWith("Two-factor authentication enabled.");

    fireEvent.click(screen.getByRole("button", { name: /I've saved them/i }));
    expect(screen.getByText("Enabled")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /disable two-factor authentication/i })).toBeInTheDocument();
  });

  it("keeps MFA on when the verification code is rejected", async () => {
    endpoints.mfaSetup.mockResolvedValue({ data: { secret: "S", otpauth_url: "otpauth://x" } });
    endpoints.mfaVerify.mockRejectedValue({ response: { status: 401 } });
    render(<Security />, { wrapper });
    fireEvent.click(screen.getByRole("button", { name: /enable two-factor authentication/i }));
    await screen.findByText("S");
    fireEvent.change(document.getElementById("mfa-verify"), { target: { value: "000000" } });
    fireEvent.click(screen.getByRole("button", { name: /verify & enable/i }));
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("That code is incorrect. Try again."));
    expect(screen.queryByText("Enabled")).toBeNull();
  });

  it("shows SSO only to owners/admins and saves the IdP connection", async () => {
    endpoints.updateSSOConnection.mockResolvedValue({
      data: { data: { configured: true, enabled: true, sp_acs_url: "https://api/sso/acs", sp_metadata_url: "https://api/sso/metadata" } },
    });
    render(<Security />, { wrapper });
    expect(await screen.findByText("Single sign-on (SAML)")).toBeInTheDocument();
    // 404 = not configured yet: the form is empty, no SP URLs to hand out.
    await waitFor(() => expect(document.getElementById("idp-entity")).toBeTruthy());
    expect(screen.queryByText("ACS (Reply) URL")).toBeNull();

    fireEvent.change(document.getElementById("idp-entity"), { target: { value: "https://idp/meta" } });
    fireEvent.change(document.getElementById("idp-sso"), { target: { value: "https://idp/sso" } });
    fireEvent.change(document.getElementById("idp-cert"), { target: { value: "-----BEGIN CERTIFICATE-----" } });
    fireEvent.click(screen.getByLabelText(/enable sso for this workspace/i));
    fireEvent.click(screen.getByRole("button", { name: /save connection/i }));

    await waitFor(() =>
      expect(endpoints.updateSSOConnection).toHaveBeenCalledWith({
        idp_entity_id: "https://idp/meta",
        idp_sso_url: "https://idp/sso",
        idp_certificate: "-----BEGIN CERTIFICATE-----",
        enabled: true,
      })
    );
    // The saved connection surfaces the SP details for the IdP.
    expect(await screen.findByText("ACS (Reply) URL")).toBeInTheDocument();
    expect(screen.getByText("https://api/sso/acs")).toBeInTheDocument();
  });

  it("hides the SSO section from members", async () => {
    authUser = { id: "u2", role: "member" };
    render(<Security />, { wrapper });
    await waitFor(() => expect(screen.getByText("Chrome · macOS")).toBeInTheDocument());
    expect(screen.queryByText("Single sign-on (SAML)")).toBeNull();
    expect(endpoints.getSSOConnection).not.toHaveBeenCalled();
  });
});
