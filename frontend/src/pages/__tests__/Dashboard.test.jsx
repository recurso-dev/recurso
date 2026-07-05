import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Dashboard from '../Dashboard';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { endpoints } from '../../lib/api';

// Mock the API module
vi.mock('../../lib/api', () => ({
    endpoints: {
        getSubscriptions: vi.fn(),
        getInvoices: vi.fn(),
        getCustomers: vi.fn(),
        getPlans: vi.fn(),
        getMRR: vi.fn()
    }
}));

// Mock Recharts since it requires ResizeObserver which isn't fully supported in jsdom
vi.mock('recharts', () => {
    const OriginalModule = vi.importActual('recharts');
    return {
        ...OriginalModule,
        ResponsiveContainer: ({ children }) => <div className="recharts-responsive-container">{children}</div>,
        AreaChart: () => <div data-testid="area-chart">AreaChart</div>,
        XAxis: () => null,
        YAxis: () => null,
        Tooltip: () => null,
        CartesianGrid: () => null,
        Area: () => null,
    };
});

// Mock Data
const mockStats = {
    netBilling: 12500.50,
    netPayments: 10200.00,
    unpaidInvoices: 2300.50,
    activeSubs: 45
};

describe('Dashboard Component', () => {
    beforeEach(() => {
        // Reset mocks
        vi.clearAllMocks();

        // Default successful response
        endpoints.getSubscriptions.mockResolvedValue({ data: { data: [{ status: 'active' }, { status: 'active' }] } }); // Mock active count logic if needed
        endpoints.getInvoices.mockResolvedValue({ data: { data: [] } });
        endpoints.getCustomers.mockResolvedValue({ data: { data: [] } });
        endpoints.getPlans.mockResolvedValue({ data: { data: [] } });
        endpoints.getMRR.mockResolvedValue({ data: { mrr: 0 } });
    });

    it('displays loading state initially', async () => {
        // Create a controllable promise
        let resolvePromise;
        const pendingPromise = new Promise((resolve) => { resolvePromise = resolve; });

        endpoints.getSubscriptions.mockReturnValue(pendingPromise);

        render(<MemoryRouter><Dashboard /></MemoryRouter>);
        expect(screen.getAllByText('...')[0]).toBeInTheDocument();

        // Resolve it
        resolvePromise({ data: { data: [] } });

        await waitFor(() => {
            expect(screen.queryByText('...')).not.toBeInTheDocument();
        });
    });

    it('displays stats correctly after loading', async () => {
        // Set specific mocks for this test
        const today = new Date().toISOString()

        endpoints.getInvoices.mockResolvedValue({
            data: {
                data: [
                    { id: 1, total: 100000, status: 'paid', created_at: today, currency: 'USD', customer_id: 'cus_1' }
                ]
            }
        });
        endpoints.getSubscriptions.mockResolvedValue({ data: { data: [] } });
        endpoints.getCustomers.mockResolvedValue({ data: { data: [] } });
        endpoints.getPlans.mockResolvedValue({ data: { data: [] } });

        render(<MemoryRouter><Dashboard /></MemoryRouter>);

        // Use findBy to wait for the element to appear
        // Wait for loading to finish
        await waitFor(() => expect(screen.queryByText('...')).not.toBeInTheDocument());

        // Check for the value (broad match)
        expect(screen.getAllByText(/1,000/).length).toBeGreaterThan(0);
    });

    it('displays empty states when no data is present', async () => {
        endpoints.getInvoices.mockResolvedValue({ data: { data: [] } });
        endpoints.getSubscriptions.mockResolvedValue({ data: { data: [] } });

        render(<MemoryRouter><Dashboard /></MemoryRouter>);

        await waitFor(() => {
            expect(screen.getByText('No recent activity')).toBeInTheDocument();
        });
    });
});
