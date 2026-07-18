import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import DunningDashboard from '../DunningDashboard';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { endpoints } from '../../lib/api';

// Tremor/recharts uses ResizeObserver, which jsdom does not implement.
global.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
};

// Mock the API module
vi.mock('../../lib/api', () => ({
    endpoints: {
        getDunningOverview: vi.fn(),
        getDunningWeights: vi.fn(),
        getDunningHistory: vi.fn(),
        getDunningRecovered: vi.fn(),
    }
}));

const currentMonth = () => {
    const d = new Date();
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
};

describe('DunningDashboard Component', () => {
    beforeEach(() => {
        vi.clearAllMocks();

        endpoints.getDunningOverview.mockResolvedValue({
            data: { total_retries: 10, total_successes: 4, success_rate: 0.4 }
        });
        endpoints.getDunningWeights.mockResolvedValue({ data: { data: [] } });
        endpoints.getDunningHistory.mockResolvedValue({ data: { data: [] } });
        endpoints.getDunningRecovered.mockResolvedValue({
            data: {
                recovered_amount_total: { INR: 236000 },
                reporting_currency: 'INR',
                reporting_total: 236000,
                recovered_count: 2,
                avg_attempts: 2.5,
                avg_days_to_recover: 4,
                monthly: [
                    { month: currentMonth(), currency: 'INR', amount: 236000, count: 2 },
                ],
            }
        });
    });

    it('displays the recovered revenue tile with totals', async () => {
        render(<MemoryRouter><DunningDashboard /></MemoryRouter>);

        await waitFor(() => {
            expect(screen.getByText('Recovered Revenue')).toBeInTheDocument();
        });

        // 236000 minor units = ₹2,360 (formatted, 0 fraction digits)
        expect(screen.getByText('₹2,360')).toBeInTheDocument();
        expect(screen.getByText('2 invoices · avg 2.5 attempts')).toBeInTheDocument();
    });

    it('renders the monthly recovered revenue chart', async () => {
        render(<MemoryRouter><DunningDashboard /></MemoryRouter>);

        await waitFor(() => {
            expect(screen.getByText('Recovered Revenue by Month')).toBeInTheDocument();
        });

        // The chart is rendered (not the empty state) when there are recoveries.
        expect(screen.getByTestId('recovered-chart')).toBeInTheDocument();
        expect(screen.queryByText(/No recovered payments yet/)).not.toBeInTheDocument();
    });

    it('shows an empty state when nothing has been recovered', async () => {
        endpoints.getDunningRecovered.mockResolvedValue({
            data: {
                recovered_amount_total: {},
                recovered_count: 0,
                avg_attempts: 0,
                avg_days_to_recover: 0,
                monthly: [],
            }
        });

        render(<MemoryRouter><DunningDashboard /></MemoryRouter>);

        await waitFor(() => {
            expect(screen.getByText('Recovered Revenue')).toBeInTheDocument();
        });
        expect(screen.getByText(/No recovered payments yet/)).toBeInTheDocument();
    });
});
