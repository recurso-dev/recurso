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

// Mock the slide-over detail (radix portal, not under test here).
vi.mock('../../components/slide-overs/CustomerDetail', () => ({
    default: () => <div data-testid="customer-detail" />
}));

const wrapper = ({ children }) => (
    <BrowserRouter>
        <ToastProvider>{children}</ToastProvider>
    </BrowserRouter>
);

const mockCustomers = [
    { id: '1', name: 'Alice Smith', email: 'alice@example.com', activeSubs: 2, risk_score: 15, created_at: '2026-01-15T00:00:00Z', tenant_id: 'tenant-1' },
    { id: '2', name: 'Bob Jones', email: 'bob@example.com', activeSubs: 0, risk_score: 55, created_at: '2026-02-01T00:00:00Z', tenant_id: 'tenant-1' },
    { id: '3', name: 'Carol Davis', email: 'carol@example.com', activeSubs: 1, risk_score: null, created_at: '2026-03-01T00:00:00Z', tenant_id: 'tenant-1' },
];

describe('Customers Page (redesign)', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('shows the skeleton loading state initially', async () => {
        let resolvePromise;
        const pending = new Promise(resolve => { resolvePromise = resolve; });
        endpoints.getCustomers.mockReturnValue(pending);

        render(<Customers />, { wrapper });

        // DataTable renders animate-pulse skeleton blocks while loading.
        expect(document.querySelector('.animate-pulse')).toBeTruthy();

        resolvePromise({ data: { data: [] } });
        await waitFor(() => {
            expect(screen.getByText('No customers yet')).toBeInTheDocument();
        });
    });

    it('shows the error state with a retry button on API failure', async () => {
        endpoints.getCustomers.mockRejectedValue(new Error('Network error'));

        render(<Customers />, { wrapper });

        await waitFor(() => {
            expect(screen.getAllByText('Network error').length).toBeGreaterThanOrEqual(1);
        });
        expect(screen.getByText('Retry')).toBeInTheDocument();

        endpoints.getCustomers.mockResolvedValue({ data: { data: mockCustomers } });
        fireEvent.click(screen.getByText('Retry'));

        await waitFor(() => {
            expect(screen.getByText('Alice Smith')).toBeInTheDocument();
        });
    });

    it('renders the customer list', async () => {
        endpoints.getCustomers.mockResolvedValue({ data: { data: mockCustomers } });

        render(<Customers />, { wrapper });

        await waitFor(() => {
            expect(screen.getByText('Alice Smith')).toBeInTheDocument();
        });
        expect(screen.getByText('alice@example.com')).toBeInTheDocument();
        expect(screen.getByText('Bob Jones')).toBeInTheDocument();
        expect(screen.getByText('Carol Davis')).toBeInTheDocument();
    });

    it('searches server-side via the q param', async () => {
        endpoints.getCustomers.mockImplementation((params = {}) => {
            const data = params.q
                ? mockCustomers.filter(c => c.name.toLowerCase().includes(params.q.toLowerCase()))
                : mockCustomers;
            return Promise.resolve({ data: { data } });
        });

        render(<Customers />, { wrapper });
        await waitFor(() => expect(screen.getByText('Alice Smith')).toBeInTheDocument());

        await userEvent.type(screen.getByPlaceholderText('Search by name or email...'), 'bob');

        await waitFor(() => {
            expect(endpoints.getCustomers).toHaveBeenCalledWith(
                expect.objectContaining({ q: 'bob' })
            );
        }, { timeout: 2000 });

        await waitFor(() => {
            expect(screen.queryByText('Alice Smith')).not.toBeInTheDocument();
        });
        expect(screen.getByText('Bob Jones')).toBeInTheDocument();
    });

    it('filters by status server-side via the status param', async () => {
        endpoints.getCustomers.mockImplementation((params = {}) => {
            let data = mockCustomers;
            if (params.status === 'inactive') data = mockCustomers.filter(c => c.activeSubs === 0);
            if (params.status === 'active') data = mockCustomers.filter(c => c.activeSubs > 0);
            return Promise.resolve({ data: { data } });
        });

        render(<Customers />, { wrapper });
        await waitFor(() => expect(screen.getByText('Alice Smith')).toBeInTheDocument());

        const buttons = screen.getAllByRole('button');
        fireEvent.click(buttons.find(b => b.textContent === 'inactive'));

        await waitFor(() => {
            expect(endpoints.getCustomers).toHaveBeenCalledWith(
                expect.objectContaining({ status: 'inactive' })
            );
        });
        await waitFor(() => {
            expect(screen.queryByText('Alice Smith')).not.toBeInTheDocument();
        });
        expect(screen.getByText('Bob Jones')).toBeInTheDocument();
    });

    it('renders risk badges', async () => {
        endpoints.getCustomers.mockResolvedValue({ data: { data: mockCustomers } });

        render(<Customers />, { wrapper });
        await waitFor(() => expect(screen.getByText('Alice Smith')).toBeInTheDocument());

        expect(screen.getByText(/15 • Low/)).toBeInTheDocument();
        expect(screen.getByText(/55 • High/)).toBeInTheDocument();
    });

    it('shows an empty state when there is no data', async () => {
        endpoints.getCustomers.mockResolvedValue({ data: { data: [] } });

        render(<Customers />, { wrapper });
        await waitFor(() => {
            expect(screen.getByText('No customers yet')).toBeInTheDocument();
        });
    });
});
