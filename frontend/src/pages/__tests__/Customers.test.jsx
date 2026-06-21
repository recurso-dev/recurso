import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Customers from '../Customers';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { endpoints } from '../../lib/api';
import { BrowserRouter } from 'react-router-dom';
import { ToastProvider } from '../../components/Toast';

// Mock API
vi.mock('../../lib/api', () => ({
    endpoints: {
        getCustomers: vi.fn(),
    }
}));

// Mock CustomerDetail to avoid import issues
vi.mock('../../components/slide-overs/CustomerDetail', () => ({
    default: () => <div data-testid="customer-detail" />
}));

const wrapper = ({ children }) => (
    <BrowserRouter>
        <ToastProvider>
            {children}
        </ToastProvider>
    </BrowserRouter>
);

const mockCustomers = [
    {
        id: '1', name: 'Alice Smith', email: 'alice@example.com',
        activeSubs: 2, risk_score: 15, created_at: '2026-01-15T00:00:00Z',
        tenant_id: 'tenant-1'
    },
    {
        id: '2', name: 'Bob Jones', email: 'bob@example.com',
        activeSubs: 0, risk_score: 55, created_at: '2026-02-01T00:00:00Z',
        tenant_id: 'tenant-1'
    },
    {
        id: '3', name: 'Carol Davis', email: 'carol@example.com',
        activeSubs: 1, risk_score: null, created_at: '2026-03-01T00:00:00Z',
        tenant_id: 'tenant-1'
    },
];

describe('Customers Page', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('shows skeleton loading state initially', async () => {
        let resolvePromise;
        const pending = new Promise(resolve => { resolvePromise = resolve; });
        endpoints.getCustomers.mockReturnValue(pending);

        render(<Customers />, { wrapper });

        // Should show the skeleton (shimmer animation divs)
        expect(document.querySelector('[style*="shimmer"]')).toBeTruthy();

        resolvePromise({ data: { data: [] } });
        await waitFor(() => {
            expect(screen.getByText('No customers yet')).toBeInTheDocument();
        });
    });

    it('shows error state with retry button on API failure', async () => {
        endpoints.getCustomers.mockRejectedValue(new Error('Network error'));

        render(<Customers />, { wrapper });

        await waitFor(() => {
            // ErrorState + Toast both render the message
            expect(screen.getAllByText('Network error').length).toBeGreaterThanOrEqual(1);
        });

        // Should show retry button in ErrorState
        expect(screen.getByText('Retry')).toBeInTheDocument();

        // Clicking retry should re-fetch
        endpoints.getCustomers.mockResolvedValue({ data: { data: mockCustomers } });
        fireEvent.click(screen.getByText('Retry'));

        await waitFor(() => {
            expect(screen.getByText('Alice Smith')).toBeInTheDocument();
        });
    });

    it('renders customer list correctly', async () => {
        endpoints.getCustomers.mockResolvedValue({ data: { data: mockCustomers } });

        render(<Customers />, { wrapper });

        await waitFor(() => {
            expect(screen.getByText('Alice Smith')).toBeInTheDocument();
        });

        expect(screen.getByText('alice@example.com')).toBeInTheDocument();
        expect(screen.getByText('Bob Jones')).toBeInTheDocument();
        expect(screen.getByText('Carol Davis')).toBeInTheDocument();
    });

    it('filters by search text', async () => {
        endpoints.getCustomers.mockResolvedValue({ data: { data: mockCustomers } });

        render(<Customers />, { wrapper });

        await waitFor(() => {
            expect(screen.getByText('Alice Smith')).toBeInTheDocument();
        });

        const searchInput = screen.getByPlaceholderText('Search by name or email...');
        await userEvent.type(searchInput, 'bob');

        expect(screen.queryByText('Alice Smith')).not.toBeInTheDocument();
        expect(screen.getByText('Bob Jones')).toBeInTheDocument();
    });

    it('filters by active/inactive status', async () => {
        endpoints.getCustomers.mockResolvedValue({ data: { data: mockCustomers } });

        render(<Customers />, { wrapper });

        await waitFor(() => {
            expect(screen.getByText('Alice Smith')).toBeInTheDocument();
        });

        // Click "Inactive" filter button (in the filter bar)
        const filterButtons = screen.getAllByRole('button');
        const inactiveBtn = filterButtons.find(b => b.textContent === 'Inactive');
        fireEvent.click(inactiveBtn);

        // Only Bob (activeSubs=0) should show
        expect(screen.queryByText('Alice Smith')).not.toBeInTheDocument();
        expect(screen.getByText('Bob Jones')).toBeInTheDocument();
        expect(screen.queryByText('Carol Davis')).not.toBeInTheDocument();

        // Click "Active" filter button 
        const activeBtn = filterButtons.find(b => b.textContent === 'Active');
        fireEvent.click(activeBtn);

        // Alice and Carol should show (activeSubs > 0)
        expect(screen.getByText('Alice Smith')).toBeInTheDocument();
        expect(screen.getByText('Carol Davis')).toBeInTheDocument();
        expect(screen.queryByText('Bob Jones')).not.toBeInTheDocument();
    });

    it('displays risk badges correctly', async () => {
        endpoints.getCustomers.mockResolvedValue({ data: { data: mockCustomers } });

        render(<Customers />, { wrapper });

        await waitFor(() => {
            expect(screen.getByText('Alice Smith')).toBeInTheDocument();
        });

        // Alice: score 15 → Low
        expect(screen.getByText(/15 • Low/)).toBeInTheDocument();
        // Bob: score 55 → High
        expect(screen.getByText(/55 • High/)).toBeInTheDocument();
    });

    it('shows empty state when no data', async () => {
        endpoints.getCustomers.mockResolvedValue({ data: { data: [] } });

        render(<Customers />, { wrapper });

        await waitFor(() => {
            expect(screen.getByText('No customers yet')).toBeInTheDocument();
        });
    });
});
