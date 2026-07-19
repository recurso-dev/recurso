import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
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
    }
}));

// Tremor's AreaChart needs ResizeObserver; stub it in jsdom.
vi.mock('@tremor/react', () => ({
    AreaChart: () => <div data-testid="area-chart" />,
}));

const renderDashboard = () =>
    render(<MemoryRouter><Dashboard /></MemoryRouter>);

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
    });

    it('renders the KPI cards after loading', async () => {
        renderDashboard();
        await waitFor(() => {
            expect(screen.getByText('MRR')).toBeInTheDocument();
        });
        expect(screen.getByText('Active Subscriptions')).toBeInTheDocument();
        expect(screen.getByText('Churn')).toBeInTheDocument();
        expect(screen.getByText('Recovered Revenue')).toBeInTheDocument();
    });

    it('shows formatted MRR, active subs and churn from the API', async () => {
        endpoints.getMRR.mockResolvedValue({ data: { mrr: 100000 } }); // $1,000.00
        endpoints.getSubscriptions.mockResolvedValue({
            data: { data: [{ status: 'active' }, { status: 'active' }, { status: 'canceled' }] },
        });

        renderDashboard();

        await waitFor(() => {
            // KPI values render as plain formatted text (consistent with Overview).
            expect(screen.getByText('$1,000.00')).toBeInTheDocument();
        });
        // 2 active subscriptions.
        expect(screen.getByText('2')).toBeInTheDocument();
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
});
