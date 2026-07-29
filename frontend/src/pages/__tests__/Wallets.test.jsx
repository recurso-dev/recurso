import { render, screen, waitFor } from "@testing-library/react";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Wallets from "../Wallets";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getWallets: vi.fn(),
    getCustomers: vi.fn().mockResolvedValue({ data: { data: [] } }),
    createWallet: vi.fn(),
    topUpWallet: vi.fn(),
    setWalletAutoRecharge: vi.fn(),
    closeWallet: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const wallets = [
  { id: "w1", customer_id: "cus_1", balance: 500000, currency: "USD" },
];

const wrapper = ({ children }) => (
  <BrowserRouter>
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
      {children}
    </QueryClientProvider>
  </BrowserRouter>
);

describe("Wallets page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getWallets.mockResolvedValue({ data: { data: wallets } });
  });

  it("renders a wallet with its balance", async () => {
    render(<Wallets />, { wrapper });
    // 500000 minor USD = $5,000.00
    await waitFor(() => expect(screen.getByText(/5,000/)).toBeInTheDocument());
  });

  it("shows the empty state with no wallets", async () => {
    endpoints.getWallets.mockResolvedValue({ data: { data: [] } });
    render(<Wallets />, { wrapper });
    await waitFor(() => expect(screen.getByText("No wallets yet")).toBeInTheDocument());
  });
});
