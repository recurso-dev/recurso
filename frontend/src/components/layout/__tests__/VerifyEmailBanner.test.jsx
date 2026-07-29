import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import VerifyEmailBanner from "../VerifyEmailBanner";
import { endpoints } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  endpoints: { resendVerification: vi.fn() },
}));
vi.mock("@/components/ui/sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

let mockUser = null;
vi.mock("@/auth/AuthProvider", () => ({
  useAuth: () => ({ user: mockUser }),
}));

describe("VerifyEmailBanner", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUser = null;
  });

  it("renders nothing when there is no user (legacy API-key session)", () => {
    mockUser = null;
    const { container } = render(<VerifyEmailBanner />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders nothing when the user's email is already verified", () => {
    mockUser = { email: "a@b.com", email_verified: true };
    const { container } = render(<VerifyEmailBanner />);
    expect(container).toBeEmptyDOMElement();
  });

  it("shows the nudge and resends the verification email when unverified", async () => {
    mockUser = { email: "diya@acme.com", email_verified: false };
    endpoints.resendVerification.mockResolvedValue({ data: {} });
    render(<VerifyEmailBanner />);

    expect(screen.getByText(/verify your email/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /resend email/i }));
    await waitFor(() =>
      expect(endpoints.resendVerification).toHaveBeenCalledTimes(1)
    );
  });

  it("can be dismissed for the session", () => {
    mockUser = { email: "diya@acme.com", email_verified: false };
    render(<VerifyEmailBanner />);
    expect(screen.getByText(/verify your email/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /dismiss/i }));
    expect(screen.queryByText(/verify your email/i)).not.toBeInTheDocument();
  });
});
