import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Invoices from '../Invoices';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { endpoints } from '../../lib/api';
import { BrowserRouter, MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('../../lib/api', () => ({
    endpoints: {
        getInvoices: vi.fn(),
        getCustomers: vi.fn(),
        sendInvoice: vi.fn(),
    }
}));

vi.mock('../../components/slide-overs/InvoiceDetail', () => ({
    default: () => <div data-testid="invoice-detail" />
}));

const wrapper = ({ children }) => (
    <BrowserRouter>
        <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
            <>
                {children}
            </>
        </QueryClientProvider>
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
        endpoints.getCustomers.mockResolvedValue({ data: { data: [] } });
    });

    // The list/filters/CSV must see the WHOLE invoice set — one max-size page
    // silently truncated all three past 250 invoices.
    it('pages through the server-paginated endpoint to load every invoice', async () => {
        const page2Invoice = {
            id: 'inv-old', invoice_number: 'INV-OLD', customer_id: 'cust-old',
            total: 1000, currency: 'USD', status: 'paid', created_at: '2025-01-01T00:00:00Z'
        };
        endpoints.getInvoices.mockImplementation(({ page }) =>
            Promise.resolve(
                page === 1
                    ? { data: { data: mockInvoices, pagination: { total_pages: 2, total: 4 } } }
                    : { data: { data: [page2Invoice], pagination: { total_pages: 2, total: 4 } } }
            )
        );
        render(<Invoices />, { wrapper });

        // Rows from BOTH pages render.
        await waitFor(() => expect(screen.getByText('INV-001')).toBeInTheDocument());
        expect(screen.getByText('INV-OLD')).toBeInTheDocument();
        expect(endpoints.getInvoices).toHaveBeenCalledWith(
            expect.objectContaining({ page: 2 })
        );
        // The full set loaded, so no truncation notice.
        expect(screen.queryByText(/Showing the newest/)).toBeNull();
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
        // Status labels are canonical (StatusBadge); the filter chips share
        // the same text, so assert the badge inside the table specifically.
        const { within } = await import('@testing-library/react');
        const table = screen.getByRole('table');
        expect(within(table).getByText('Paid')).toBeInTheDocument();
        expect(within(table).getByText('Open')).toBeInTheDocument();
        expect(within(table).getByText('Void')).toBeInTheDocument();
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

    it('filters by status chip', async () => {
        endpoints.getInvoices.mockResolvedValue({ data: { data: mockInvoices } });

        render(<Invoices />, { wrapper });

        await waitFor(() => {
            expect(screen.getByText('INV-001')).toBeInTheDocument();
        });

        // Click the "Paid" status chip — only the paid invoice remains.
        await userEvent.click(screen.getByRole('button', { name: 'Paid' }));
        expect(screen.getByText('INV-001')).toBeInTheDocument(); // paid
        expect(screen.queryByText('INV-002')).not.toBeInTheDocument(); // open
        expect(screen.queryByText('INV-003')).not.toBeInTheDocument(); // void
    });

    it('offers a CSV export action', async () => {
        endpoints.getInvoices.mockResolvedValue({ data: { data: mockInvoices } });

        render(<Invoices />, { wrapper });

        await waitFor(() => {
            expect(screen.getByText('INV-001')).toBeInTheDocument();
        });
        expect(screen.getByRole('button', { name: /export csv/i })).toBeEnabled();
    });

    it('filters to one aging bucket when deep-linked with ?aging=', async () => {
        const day = 86400000;
        endpoints.getInvoices.mockResolvedValue({
            data: {
                data: [
                    // 45 days overdue and unpaid → the 31-60 bucket.
                    { id: 'inv-old', invoice_number: 'INV-OLD', customer_id: 'c1', total: 10000, amount_paid: 0, currency: 'USD', status: 'past_due', due_date: new Date(Date.now() - 45 * day).toISOString(), created_at: '2026-06-01T00:00:00Z' },
                    // Due tomorrow → current, filtered out.
                    { id: 'inv-fresh', invoice_number: 'INV-FRESH', customer_id: 'c2', total: 5000, amount_paid: 0, currency: 'USD', status: 'open', due_date: new Date(Date.now() + day).toISOString(), created_at: '2026-08-01T00:00:00Z' },
                ],
            },
        });

        render(<Invoices />, {
            wrapper: ({ children }) => (
                <MemoryRouter initialEntries={['/invoices?aging=31-60']}>
                    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
                        {children}
                    </QueryClientProvider>
                </MemoryRouter>
            ),
        });

        await waitFor(() => expect(screen.getByText('INV-OLD')).toBeInTheDocument());
        expect(screen.queryByText('INV-FRESH')).not.toBeInTheDocument();
        // The active bucket shows as a clearable chip.
        expect(screen.getByRole('button', { name: /Overdue 31–60 days/ })).toBeInTheDocument();
    });

    const loadThree = () =>
        endpoints.getInvoices.mockResolvedValue({
            data: { data: mockInvoices, pagination: { total_pages: 1, total: 3 } },
        });

    it('bulk-sends the selected invoices with a per-record idempotency key', async () => {
        loadThree();
        endpoints.sendInvoice.mockResolvedValue({ data: {} });
        const user = userEvent.setup();
        render(<Invoices />, { wrapper });
        await waitFor(() => expect(screen.getByText('INV-001')).toBeInTheDocument());

        // Select all on this page → bulk bar with the scoped action.
        await user.click(screen.getByRole('checkbox', { name: 'Select all rows on this page' }));
        await user.click(screen.getByRole('button', { name: 'Send 3 invoices' }));
        // Confirm dialog states the exact scope.
        const dialog = await screen.findByRole('dialog');
        await user.click(within(dialog).getByRole('button', { name: 'Send 3 invoices' }));

        await waitFor(() => expect(endpoints.sendInvoice).toHaveBeenCalledTimes(3));
        // Each call carries a distinct idempotency key.
        const keys = endpoints.sendInvoice.mock.calls.map(([, opts]) => opts.idempotencyKey);
        expect(new Set(keys).size).toBe(3);
        expect(keys.every(Boolean)).toBe(true);
        await waitFor(() => expect(screen.getByText(/All 3 invoices done/)).toBeInTheDocument());
    });

    it('surfaces a partial failure and retries only the failed record with the same key', async () => {
        loadThree();
        endpoints.sendInvoice.mockImplementation((id) =>
            id === 'inv-3'
                ? Promise.reject({ response: { data: { error: { message: 'cannot send a void invoice' } } } })
                : Promise.resolve({ data: {} })
        );
        const user = userEvent.setup();
        render(<Invoices />, { wrapper });
        await waitFor(() => expect(screen.getByText('INV-001')).toBeInTheDocument());

        await user.click(screen.getByRole('checkbox', { name: 'Select all rows on this page' }));
        await user.click(screen.getByRole('button', { name: 'Send 3 invoices' }));
        const dialog = await screen.findByRole('dialog');
        await user.click(within(dialog).getByRole('button', { name: 'Send 3 invoices' }));

        // Partial failure is its own state — never reported as success.
        await waitFor(() => expect(screen.getByText('Partially failed')).toBeInTheDocument());
        const resultDialog = screen.getByRole('dialog');
        expect(within(resultDialog).getByText('2 succeeded, 1 failed.')).toBeInTheDocument();
        // The failed record stays identifiable in the result surface.
        expect(within(resultDialog).getByText('INV-003')).toBeInTheDocument();
        expect(within(resultDialog).getByText('cannot send a void invoice')).toBeInTheDocument();

        const keyForVoid = endpoints.sendInvoice.mock.calls.find(([id]) => id === 'inv-3')[1].idempotencyKey;

        // Retry only the failed one; let it succeed this time.
        endpoints.sendInvoice.mockResolvedValue({ data: {} });
        await user.click(screen.getByRole('button', { name: 'Retry 1 failed' }));
        await waitFor(() => expect(screen.getByText(/All 1 invoice done/)).toBeInTheDocument());

        // The retry reused the same idempotency key (no double-act) and only re-ran inv-3.
        const voidCalls = endpoints.sendInvoice.mock.calls.filter(([id]) => id === 'inv-3');
        expect(voidCalls).toHaveLength(2);
        expect(voidCalls[1][1].idempotencyKey).toBe(keyForVoid);
    });
});
