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
        getEvents: vi.fn(),
        getMRRWaterfall: vi.fn(),
        runReconciliation: vi.fn(),
        getCollectionsFunnel: vi.fn(),
        getCollectionsQueue: vi.fn(),
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
        endpoints.getEvents.mockResolvedValue({ data: { data: [] } });
        endpoints.getMRRWaterfall.mockResolvedValue({ data: { data: null } });
        endpoints.getInvoiceAging.mockResolvedValue({
            data: { data: { reporting_currency: 'USD', buckets: [], total_outstanding: 0, total_count: 0 } },
        });
        endpoints.runReconciliation.mockResolvedValue({ data: { data: { total_discrepancies: 0 } } });
        endpoints.getCollectionsFunnel.mockResolvedValue({
            data: { data: { reporting_currency: 'USD', past_due: { count: 0, amount: 0 }, fx_excluded_currencies: [] } },
        });
        endpoints.getCollectionsQueue.mockResolvedValue({ data: { data: [], meta: { total: 0 } } });
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

    it('itemizes failing payments in the needs-attention surface, each drilling to its invoice', async () => {
        endpoints.getCollectionsFunnel.mockResolvedValue({
            data: {
                data: {
                    reporting_currency: 'USD',
                    past_due: { count: 2, amount: 90000 },
                    fx_excluded_currencies: [],
                },
            },
        });
        endpoints.getCollectionsQueue.mockResolvedValue({
            data: {
                data: [
                    {
                        id: 'inv_od1',
                        customer_name: 'Acme Corp',
                        invoice_number: 'INV-001',
                        currency: 'USD',
                        amount_remaining: 50000,
                        last_payment_error: 'insufficient_funds',
                        days_overdue: 4,
                    },
                    {
                        id: 'inv_od2',
                        customer_name: 'Globex',
                        invoice_number: 'INV-002',
                        currency: 'USD',
                        amount_remaining: 40000,
                        last_payment_error: 'card_declined',
                        days_overdue: 9,
                    },
                ],
                meta: { total: 2 },
            },
        });

        renderDashboard();

        await waitFor(() => {
            expect(screen.getByText('2 invoices failing collection')).toBeInTheDocument();
        });
        // Revenue at risk, FX-normalized.
        expect(screen.getByText(/\$900\.00 at risk/)).toBeInTheDocument();
        // Each row names WHAT (customer + invoice), WHY (humanized failure), and
        // drills to WHAT object (the invoice).
        const acmeRow = screen.getByText('Acme Corp').closest('a');
        expect(acmeRow).toHaveAttribute('href', '/invoices/inv_od1');
        expect(within(acmeRow).getByText(/Insufficient funds/)).toBeInTheDocument();
        expect(within(acmeRow).getByText(/4d overdue/)).toBeInTheDocument();
        // "View all" drills to the canonical Collections list (no Home-specific page).
        const viewAllToCollections = screen
            .getAllByText('View all')
            .map((el) => el.closest('a'))
            .find((a) => a?.getAttribute('href') === '/collections');
        expect(viewAllToCollections).toBeTruthy();
    });

    it('shows an honest "showing N of M" count and does not imply the whole list is on Home', async () => {
        endpoints.getCollectionsFunnel.mockResolvedValue({
            data: {
                data: {
                    reporting_currency: 'USD',
                    past_due: { count: 37, amount: 500000 },
                    fx_excluded_currencies: ['EUR'],
                },
            },
        });
        endpoints.getCollectionsQueue.mockResolvedValue({
            data: {
                data: [
                    { id: 'inv_a', customer_name: 'A', invoice_number: 'INV-A', currency: 'USD', amount_remaining: 1000, last_payment_error: 'card_declined', days_overdue: 2 },
                ],
                meta: { total: 37 },
            },
        });

        renderDashboard();

        await waitFor(() => {
            expect(screen.getByText('37 invoices failing collection')).toBeInTheDocument();
        });
        // Count honesty: the header reflects the true total, the footer says how
        // many are actually shown, and the FX caveat is surfaced.
        expect(screen.getByText(/Showing 1 of 37/)).toBeInTheDocument();
        expect(screen.getByText(/excludes EUR/)).toBeInTheDocument();
    });

    it('shows "data unavailable" for payments — never a false all-clear — when the collections fetch fails', async () => {
        endpoints.getCollectionsFunnel.mockRejectedValue(new Error('down'));
        endpoints.getCollectionsQueue.mockRejectedValue(new Error('down'));

        renderDashboard();

        await waitFor(() => {
            expect(screen.getByText(/Payments in recovery — data unavailable/)).toBeInTheDocument();
        });
        // A failed source must not resolve into "all caught up".
        expect(screen.queryByText(/all caught up/i)).not.toBeInTheDocument();
    });

    it('surfaces ledger reconciliation discrepancies in the needs-attention strip', async () => {
        endpoints.runReconciliation.mockResolvedValue({
            data: { data: { total_discrepancies: 3 } },
        });

        renderDashboard();

        await waitFor(() => {
            expect(screen.getByText('3 reconciliation discrepancies')).toBeInTheDocument();
        });
        // The tile drills to the reconciliation page.
        expect(screen.getByText('3 reconciliation discrepancies').closest('a')).toHaveAttribute(
            'href',
            '/finance/reconciliation'
        );
    });

    it('shows the "all caught up" line only when every source settled with nothing to do', async () => {
        renderDashboard();
        await waitFor(() => {
            expect(screen.getByText(/all caught up/i)).toBeInTheDocument();
        });
        // No payments card renders when the queue is genuinely empty.
        expect(screen.queryByText(/failing collection/)).not.toBeInTheDocument();
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
                        status: 'Paid',
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
        expect(screen.getByText('Paid')).toBeInTheDocument();
    });

    it('renders the activity feed with links to each event object', async () => {
        endpoints.getEvents.mockResolvedValue({
            data: {
                data: [
                    { id: 'ev1', type: 'invoice.paid', object_type: 'invoice', object_id: 'inv_9', created_at: '2026-08-12T10:00:00Z' },
                    { id: 'ev2', type: 'subscription.created', object_type: 'subscription', object_id: 'sub_9', created_at: '2026-08-12T09:00:00Z' },
                ],
            },
        });
        renderDashboard();
        await waitFor(() => expect(screen.getByText('Invoice paid')).toBeInTheDocument());
        expect(screen.getByText('Invoice paid').closest('a')).toHaveAttribute('href', '/invoices/inv_9');
        expect(screen.getByText('Subscription created').closest('a')).toHaveAttribute('href', '/subscriptions/sub_9');
    });

    it('shows an honest MRR delta from the waterfall (starting → ending)', async () => {
        endpoints.getMRR.mockResolvedValue({ data: { mrr: 110000, reporting_currency: 'USD' } });
        endpoints.getMRRWaterfall.mockResolvedValue({
            data: { data: { starting_mrr: 100000, ending_mrr: 110000 } },
        });
        renderDashboard();
        await waitFor(() => expect(screen.getByText('+10%')).toBeInTheDocument());
        expect(screen.getByText('vs 30 days ago')).toBeInTheDocument();
    });

    it('shows no MRR delta when there is no starting base to compare against', async () => {
        endpoints.getMRRWaterfall.mockResolvedValue({
            data: { data: { starting_mrr: 0, ending_mrr: 110000 } },
        });
        renderDashboard();
        await waitFor(() => expect(screen.getByText('MRR')).toBeInTheDocument());
        expect(screen.queryByText('vs 30 days ago')).not.toBeInTheDocument();
    });

    it('shows the 30-day new-subscription comparison on the Active Subscriptions card', async () => {
        const now = Date.now();
        const iso = (daysAgo) => new Date(now - daysAgo * 86_400_000).toISOString();
        endpoints.getSubscriptions.mockResolvedValue({
            data: {
                data: [
                    // two new this window, one in the prior window → +100%
                    { id: 's1', status: 'active', created_at: iso(5) },
                    { id: 's2', status: 'active', created_at: iso(10) },
                    { id: 's3', status: 'active', created_at: iso(45) },
                ],
            },
        });
        renderDashboard();
        await waitFor(() => expect(screen.getByText('2 new in last 30 days')).toBeInTheDocument());
        expect(screen.getByText('+100%')).toBeInTheDocument();
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
