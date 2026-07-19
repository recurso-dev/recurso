import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Invoices from '../Invoices';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { endpoints } from '../../lib/api';
import { BrowserRouter } from 'react-router-dom';
import { ToastProvider } from '../../components/Toast';

vi.mock('../../lib/api', () => ({
    endpoints: {
        getInvoices: vi.fn(),
    }
}));

vi.mock('../../components/slide-overs/InvoiceDetail', () => ({
    default: () => <div data-testid="invoice-detail" />
}));

const wrapper = ({ children }) => (
    <BrowserRouter>
        <ToastProvider>
            {children}
        </ToastProvider>
    </BrowserRouter>
);

const mockInvoices = [
    {
        id: 'inv-1', invoice_number: 'INV-001', customer_id: 'cust-12345678-abcd',
        total: 150000, currency: 'USD', status: 'paid', created_at: '2026-01-15T00:00:00Z'
    },
    {
        id: 'inv-2', invoice_number: 'INV-002', customer_id: 'cust-87654321-efgh',
        total: 50000, currency: 'USD', status: 'open', created_at: '2026-02-01T00:00:00Z'
    },
    {
        id: 'inv-3', invoice_number: 'INV-003', customer_id: 'cust-11111111-ijkl',
        total: 0, currency: 'INR', status: 'void', created_at: '2026-03-01T00:00:00Z'
    },
];

describe('Invoices Page', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('shows loading skeleton initially', async () => {
        let resolvePromise;
        const pending = new Promise(resolve => { resolvePromise = resolve; });
        endpoints.getInvoices.mockReturnValue(pending);

        render(<Invoices />, { wrapper });
        expect(document.querySelector('.animate-pulse')).toBeTruthy();

        resolvePromise({ data: { data: [] } });
        await waitFor(() => {
            expect(screen.getByText('No invoices yet')).toBeInTheDocument();
        });
    });

    it('shows error state with retry on API failure', async () => {
        endpoints.getInvoices.mockRejectedValue(new Error('Server error'));

        render(<Invoices />, { wrapper });

        await waitFor(() => {
            // ErrorState + Toast both render the message
            expect(screen.getAllByText('Server error').length).toBeGreaterThanOrEqual(1);
        });

        expect(screen.getByText('Retry')).toBeInTheDocument();
    });

    it('renders invoices with correct formatting', async () => {
        endpoints.getInvoices.mockResolvedValue({ data: { data: mockInvoices } });

        render(<Invoices />, { wrapper });

        await waitFor(() => {
            expect(screen.getByText('INV-001')).toBeInTheDocument();
        });

        // Check amount formatting
        expect(
            screen.getByText((_, el) => el?.classList?.contains("money") && el.textContent === "$1,500.00")
        ).toBeInTheDocument();
        expect(
            screen.getByText((_, el) => el?.classList?.contains("money") && el.textContent === "$500.00")
        ).toBeInTheDocument();

        // Check status badges
        expect(screen.getByText('paid')).toBeInTheDocument();
        expect(screen.getByText('open')).toBeInTheDocument();
        expect(screen.getByText('void')).toBeInTheDocument();
    });

    it('search filters by invoice number', async () => {
        endpoints.getInvoices.mockResolvedValue({ data: { data: mockInvoices } });

        render(<Invoices />, { wrapper });

        await waitFor(() => {
            expect(screen.getByText('INV-001')).toBeInTheDocument();
        });

        const searchInput = screen.getByPlaceholderText('Search invoices...');
        await userEvent.type(searchInput, 'INV-002');

        expect(screen.queryByText('INV-001')).not.toBeInTheDocument();
        expect(screen.getByText('INV-002')).toBeInTheDocument();
    });

    it('shows empty state when no invoices', async () => {
        endpoints.getInvoices.mockResolvedValue({ data: { data: [] } });

        render(<Invoices />, { wrapper });

        await waitFor(() => {
            expect(screen.getByText('No invoices yet')).toBeInTheDocument();
        });
    });
});
