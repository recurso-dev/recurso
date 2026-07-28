import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi, beforeEach } from 'vitest';

import Entities from '../Entities';
import { endpoints } from '../../lib/api';

vi.mock('../../lib/api', () => ({
    endpoints: { getEntitiesOverview: vi.fn() },
}));

const wrap = (ui) => (
    <MemoryRouter>
        <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
            {ui}
        </QueryClientProvider>
    </MemoryRouter>
);

describe('Entities overview page', () => {
    beforeEach(() => vi.clearAllMocks());

    it('renders per-entity MRR + AR for a multi-entity tenant', async () => {
        endpoints.getEntitiesOverview.mockResolvedValue({
            data: {
                data: {
                    reporting_currency: 'USD',
                    total_mrr: 400000,
                    total_ar_outstanding: 190000,
                    entities: [
                        { entity_id: 'b', entity_name: 'Entity B', is_primary: false, mrr: 300000, arr: 3600000, ar_outstanding: 120000, subscriptions: 12 },
                        { entity_id: 'a', entity_name: 'HQ', is_primary: true, mrr: 100000, arr: 1200000, ar_outstanding: 70000, subscriptions: 4 },
                    ],
                },
            },
        });

        render(wrap(<Entities />));

        await waitFor(() => expect(screen.getByText('Entity B')).toBeInTheDocument());
        expect(screen.getByText('HQ')).toBeInTheDocument();
        expect(screen.getByText('primary')).toBeInTheDocument();
        // Summary stat.
        expect(screen.getByText('Legal entities')).toBeInTheDocument();
    });

    it('shows a single-entity pointer when there is only one entity', async () => {
        endpoints.getEntitiesOverview.mockResolvedValue({
            data: {
                data: {
                    reporting_currency: 'USD',
                    total_mrr: 100000,
                    total_ar_outstanding: 0,
                    entities: [{ entity_id: 'a', entity_name: 'HQ', is_primary: true, mrr: 100000, arr: 1200000, ar_outstanding: 0, subscriptions: 4 }],
                },
            },
        });

        render(wrap(<Entities />));

        await waitFor(() => expect(screen.getByText('Single-entity account')).toBeInTheDocument());
    });
});
