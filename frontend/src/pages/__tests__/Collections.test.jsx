import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi, beforeEach } from 'vitest';

import Collections from '../Collections';
import { endpoints } from '../../lib/api';

vi.mock('../../lib/api', () => ({
    endpoints: {
        getCollectionsQueue: vi.fn(),
        getCollectionsFunnel: vi.fn(),
        getCollectionsFailures: vi.fn(),
    },
}));

const funnelResponse = {
    data: {
        data: {
            reporting_currency: 'USD',
            past_due: { count: 4, amount: 40000 },
            uncollectible: { count: 2, amount: 15000 },
            recovered: { count: 6, amount: 30000 },
            recovery_rate: 0.75,
        },
    },
};
const failuresResponse = {
    data: {
        data: [
            { error_code: 'card_declined', count: 5, amount_at_risk: 20000 },
            { error_code: 'insufficient_funds', count: 3, amount_at_risk: 8000 },
        ],
    },
};

const wrap = (ui) => (
    <MemoryRouter>
        <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
            {ui}
        </QueryClientProvider>
    </MemoryRouter>
);

describe('Collections page', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        endpoints.getCollectionsFunnel.mockResolvedValue(funnelResponse);
        endpoints.getCollectionsFailures.mockResolvedValue(failuresResponse);
    });

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
        // Humanized codes, not the raw snake_case (may appear in both the table
        // row and the failure-breakdown card).
        expect(screen.getAllByText('Insufficient funds').length).toBeGreaterThanOrEqual(1);
        expect(screen.getByText('ACH return')).toBeInTheDocument();
        // Status badges (label also appears as a filter tab, hence getAllByText).
        expect(screen.getAllByText('Past due').length).toBeGreaterThanOrEqual(1);
        expect(screen.getAllByText('Uncollectible').length).toBeGreaterThanOrEqual(1);
        // Returned ACH attempt chip.
        expect(screen.getByText('returned')).toBeInTheDocument();
        expect(screen.getByText('Globex')).toBeInTheDocument();

        // Funnel stats + failure breakdown (tenant-wide, FX-normalized).
        expect(screen.getByText('Recovery rate')).toBeInTheDocument();
        expect(screen.getByText('75.0%')).toBeInTheDocument();
        expect(screen.getByText('Top failure reasons')).toBeInTheDocument();
        // "Card declined" appears both as a failure-breakdown row and (humanized)
        // in the table, so assert at least one.
        expect(screen.getAllByText('Card declined').length).toBeGreaterThanOrEqual(1);

        // Each row exposes a manual-actions menu (Inc 3), one per invoice.
        expect(screen.getAllByLabelText('Invoice actions')).toHaveLength(2);
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

    it('surfaces a retryable error instead of $0 KPIs when analytics fails', async () => {
        endpoints.getCollectionsQueue.mockResolvedValue({
            data: { data: [], meta: { page: 1, per_page: 25, total: 0 } },
        });
        endpoints.getCollectionsFunnel.mockRejectedValue(new Error('boom'));

        render(wrap(<Collections />));

        await waitFor(() =>
            expect(screen.getByText(/couldn't load collections analytics/i)).toBeInTheDocument()
        );
        // Never render the misleading "$0.00 revenue at risk" on a failed fetch.
        expect(screen.queryByText('Revenue at risk')).not.toBeInTheDocument();
        expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument();
    });
});
