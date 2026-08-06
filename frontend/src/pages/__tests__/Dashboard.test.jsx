import { render, screen, waitFor, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import Dashboard from '../Dashboard';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { endpoints } from '../../lib/api';

// Mock the API module (cockpit wires MRR + dunning recovered + lists +
// the needs-attention feeds).
vi.mock('../../lib/api', () => ({
    endpoints: {
        getSubscriptions: vi.fn(),
        getInvoices: vi.fn(),
        getCustomers: vi.fn(),
        getMRR: vi.fn(),
        getDunningRecovered: vi.fn(),
        getDisputes: vi.fn(),
        getChurnAlerts: vi.fn(),
        getInvoiceAging: vi.fn(),
    }
}));

// Tremor charts need ResizeObserver; stub the ones the page uses in jsdom.
vi.mock('@tremor/react', () => ({
    AreaChart: () => <div data-testid="area-chart" />,
    DonutChart: () => <div data-testid="donut-chart" />,
}));

const renderDashboard = () =>
    render(
        <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
            <MemoryRouter><Dashboard /></MemoryRouter>
        </QueryClientProvider>
    );

describe('Dashboard (redesign)', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        endpoints.getSubscriptions.mockResolvedValue({ data: { data: [] } });
        endpoints.getInvoices.mockResolvedValue({ data: { data: [] } });
        endpoints.getCustomers.mockResolvedValue({ data: { data: [] } });
        endpoints.getMRR.mockResolvedValue({ data: { mrr: 0 } });
        endpoints.getDunningRecovered.mockResolvedValue({ data: { recovered: 0 } });
        endpoints.getDisputes.mockResolvedValue({ data: { data: [] } });
        endpoints.getChurnAlerts.mockResolvedValue({ data: { data: [] } });
        endpoints.getInvoiceAging.mockResolvedValue({
            data: { data: { reporting_currency: 'USD', buckets: [], total_outstanding: 0, total_count: 0 } },
        });
    });

    it('renders the KPI cards after loading', async () => {
        renderDashboard();
        await waitFor(() => {
            expect(screen.getByText('MRR')).toBeInTheDocument();
        });
        expect(screen.getByText('Active Subscriptions')).toBeInTheDocument();
        expect(screen.getByText('Churn')).toBeInTheDocument();
        expect(screen.getByText('Recovered Revenue')).toBeInTheDocument();
        // Ambiguous metrics carry a plain-language definition (native title on
        // these linked tiles).
        expect(screen.getByText('Churn')).toHaveAttribute('title', expect.stringContaining('not revenue-weighted'));
    });

    it('shows formatted MRR, active subs and churn from the API', async () => {
        endpoints.getMRR.mockResolvedValue({ data: { mrr: 100000 } }); // $1,000
        endpoints.getSubscriptions.mockResolvedValue({
            data: { data: [{ status: 'active' }, { status: 'active' }, { status: 'canceled' }] },
        });

        renderDashboard();

        await waitFor(() => {
            // Headline KPI formatting drops the ".00" tail on whole amounts.
            expect(screen.getByText('$1,000')).toBeInTheDocument();
        });
        // 2 active subscriptions — assert via the KPI tile (the subscription-mix
        // legend also shows "2", so scope the match to the "Active Subscriptions"
        // card rather than a bare getByText).
        const activeCard = screen.getByText('Active Subscriptions').closest('a');
        expect(within(activeCard).getByText('2')).toBeInTheDocument();
        // Churn = 1 canceled / 3 total = 33.3%.
        expect(screen.getByText('33.3%')).toBeInTheDocument();
    });

    it('surfaces overdue invoices in the needs-attention strip', async () => {
        endpoints.getInvoices.mockResolvedValue({
            data: {
                data: [
                    {
                        id: 'inv_od',
                        total: 50000,
                        amount_due: 50000,
                        status: 'past_due',
                        currency: 'USD',
                        customer_id: 'cus_1',
                        created_at: new Date().toISOString(),
                    },
                ],
            },
        });

        renderDashboard();

        await waitFor(() => {
            expect(screen.getByText('1 overdue invoice')).toBeInTheDocument();
        });
        // The card links to the aging report.
        expect(screen.getByText('1 overdue invoice').closest('a')).toHaveAttribute(
            'href',
            '/finance/invoice-aging'
        );
    });

    it('shows the all-clear line when nothing needs attention', async () => {
        renderDashboard();
        await waitFor(() => {
            expect(screen.getByText(/All clear/)).toBeInTheDocument();
        });
    });

    it('shows a graceful empty state when there are no invoices', async () => {
        renderDashboard();
        await waitFor(() => {
            expect(screen.getByText('No revenue yet')).toBeInTheDocument();
        });
    });

    it('renders a recent invoice with a status badge', async () => {
        endpoints.getInvoices.mockResolvedValue({
            data: {
                data: [
                    {
                        id: 'inv_1',
                        total: 25000,
                        status: 'paid',
                        currency: 'USD',
                        customer_id: 'cus_1',
                        created_at: new Date().toISOString(),
                    },
                ],
            },
        });
        endpoints.getCustomers.mockResolvedValue({
            data: { data: [{ id: 'cus_1', name: 'Acme Corp' }] },
        });

        renderDashboard();

        await waitFor(() => {
            expect(screen.getByText('Acme Corp')).toBeInTheDocument();
        });
        expect(
          screen.getByText((_, el) => el?.classList?.contains("money") && el.textContent === "$250.00")
        ).toBeInTheDocument();
        expect(screen.getByText('paid')).toBeInTheDocument();
    });

    it('shows a retryable error (not a page of zeros) when every core read fails', async () => {
        // Simulate a total outage: the core reads all reject. An empty tenant
        // (data: []) must NOT trip this — only genuine failures do.
        endpoints.getSubscriptions.mockRejectedValue(new Error('down'));
        endpoints.getInvoices.mockRejectedValue(new Error('down'));
        endpoints.getCustomers.mockRejectedValue(new Error('down'));
        endpoints.getMRR.mockRejectedValue(new Error('down'));

        renderDashboard();

        await waitFor(() =>
            expect(screen.getByText(/couldn't load your dashboard/i)).toBeInTheDocument()
        );
        expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument();
        // The misleading $0 KPI tiles must not render.
        expect(screen.queryByText('MRR')).not.toBeInTheDocument();
        expect(screen.queryByText('Active Subscriptions')).not.toBeInTheDocument();
    });

    it('still renders the dashboard for a genuinely empty tenant (no error)', async () => {
        // All reads succeed but return empty — this is a real, empty account,
        // not an outage, so the KPI tiles must show (as zeros), not an error.
        renderDashboard();
        await waitFor(() => expect(screen.getByText('MRR')).toBeInTheDocument());
        expect(screen.queryByText(/couldn't load your dashboard/i)).not.toBeInTheDocument();
    });
});
