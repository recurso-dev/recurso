import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi, beforeEach } from 'vitest';

import Collections from '../Collections';
import { endpoints } from '../../lib/api';

vi.mock('../../lib/api', () => ({
    endpoints: {
        getCollectionsQueue: vi.fn(),
    },
}));

const wrap = (ui) => (
    <MemoryRouter>
        <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
            {ui}
        </QueryClientProvider>
    </MemoryRouter>
);

describe('Collections page', () => {
    beforeEach(() => vi.clearAllMocks());

    it('renders failing invoices with humanized failure reasons', async () => {
        endpoints.getCollectionsQueue.mockResolvedValue({
            data: {
                data: [
                    {
                        id: 'inv-1',
                        customer_id: 'cust-1',
                        customer_name: 'Acme Inc',
                        customer_email: 'billing@acme.com',
                        invoice_number: 'INV-0001',
                        status: 'past_due',
                        currency: 'USD',
                        amount_remaining: 12500,
                        due_date: new Date().toISOString(),
                        days_overdue: 12,
                        retry_count: 2,
                        last_payment_error: 'insufficient_funds',
                        next_retry_at: new Date(Date.now() + 3600_000).toISOString(),
                        managed_by: 'worker',
                        attempt_status: '',
                    },
                    {
                        id: 'inv-2',
                        customer_id: 'cust-2',
                        customer_name: 'Globex',
                        customer_email: 'ap@globex.com',
                        invoice_number: 'INV-0002',
                        status: 'uncollectible',
                        currency: 'USD',
                        amount_remaining: 9900,
                        due_date: new Date().toISOString(),
                        days_overdue: 45,
                        retry_count: 10,
                        last_payment_error: 'ach_return',
                        next_retry_at: null,
                        managed_by: 'scheduler',
                        attempt_status: 'returned',
                    },
                ],
                meta: { page: 1, per_page: 25, total: 2 },
            },
        });

        render(wrap(<Collections />));

        await waitFor(() => expect(screen.getByText('Acme Inc')).toBeInTheDocument());
        // Humanized codes, not the raw snake_case.
        expect(screen.getByText('Insufficient funds')).toBeInTheDocument();
        expect(screen.getByText('ACH return')).toBeInTheDocument();
        // Status badges (label also appears as a filter tab, hence getAllByText).
        expect(screen.getAllByText('Past due').length).toBeGreaterThanOrEqual(1);
        expect(screen.getAllByText('Uncollectible').length).toBeGreaterThanOrEqual(1);
        // Returned ACH attempt chip.
        expect(screen.getByText('returned')).toBeInTheDocument();
        // Total count in the header stat.
        expect(screen.getByText('Globex')).toBeInTheDocument();
    });

    it('shows the good-news empty state when nothing is failing', async () => {
        endpoints.getCollectionsQueue.mockResolvedValue({
            data: { data: [], meta: { page: 1, per_page: 25, total: 0 } },
        });

        render(wrap(<Collections />));

        await waitFor(() =>
            expect(screen.getByText('Nothing in collections')).toBeInTheDocument()
        );
    });
});
