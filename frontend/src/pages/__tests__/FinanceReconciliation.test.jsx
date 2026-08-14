import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import FinanceReconciliation from '../FinanceReconciliation';
import { endpoints } from '../../lib/api';

// Mock the API module
vi.mock('../../lib/api', () => ({
    endpoints: {
        runReconciliation: vi.fn(),
        recordReconciliation: vi.fn(),
        getReconciliationRuns: vi.fn(),
    }
}));

const balancedReport = {
    tenant_id: 'ten-1',
    started_at: '2026-07-06T10:00:00Z',
    finished_at: '2026-07-06T10:00:01Z',
    invoices_checked: 42,
    paid_invoices_checked: 30,
    total_discrepancies: 0,
    discrepancies: [],
    truncated: false,
    tb_compared: true,
    tb_accounts_checked: 4,
    tb_transfers_checked: 120,
};

const driftReport = {
    ...balancedReport,
    total_discrepancies: 2,
    discrepancies: [
        {
            type: 'invoice_amount_mismatch',
            invoice_id: 'aaaaaaaa-1111-2222-3333-444444444444',
            expected_amount: 5000,
            found_amount: 4500,
        },
        {
            type: 'missing_in_tigerbeetle',
            transaction_id: 'bbbbbbbb-1111-2222-3333-444444444444',
            expected_amount: 900,
            found_amount: 0,
        },
    ],
    tb_compared: false,
    tb_skip_reason: 'TigerBeetle client is not connected',
};

const renderPage = () =>
    render(
        <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
            <MemoryRouter>
                <FinanceReconciliation />
            </MemoryRouter>
        </QueryClientProvider>
    );

describe('FinanceReconciliation page', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        endpoints.getReconciliationRuns.mockResolvedValue({ data: { data: [] } });
        endpoints.recordReconciliation.mockResolvedValue({ data: { data: balancedReport } });
    });

    it('renders summary cards and the discrepancy table', async () => {
        endpoints.runReconciliation.mockResolvedValue({ data: { data: driftReport } });
        renderPage();

        await waitFor(() => {
            expect(screen.getByText('Invoices Checked')).toBeInTheDocument();
        });

        expect(screen.getByText('42')).toBeInTheDocument();
        expect(screen.getByText('30 paid invoices')).toBeInTheDocument();
        expect(screen.getByText('Invoice amount mismatch')).toBeInTheDocument();
        expect(screen.getByText('Missing in TigerBeetle')).toBeInTheDocument();
        expect(screen.getByText('5,000')).toBeInTheDocument();
        expect(screen.getByText('4,500')).toBeInTheDocument();
        // The invoice id drills through to that invoice's detail on the
        // Invoices page (not a dead UUID).
        const invoiceLink = screen.getByRole('link', { name: 'aaaaaaaa…' });
        expect(invoiceLink).toHaveAttribute('href', '/invoices/aaaaaaaa-1111-2222-3333-444444444444');
        // Depth: the difference (found − expected) is shown with a sign, and each
        // row explains WHY, not just a raw enum. And the verdict headline names
        // the count.
        expect(screen.getByText('-500')).toBeInTheDocument();
        expect(screen.getByText(/issuance posting doesn.t equal the invoice total/i)).toBeInTheDocument();
        expect(screen.getByText(/2 discrepancies to resolve/i)).toBeInTheDocument();
    });

    it('shows the skipped TigerBeetle badge with the skip reason', async () => {
        endpoints.runReconciliation.mockResolvedValue({ data: { data: driftReport } });
        renderPage();

        await waitFor(() => {
            expect(screen.getByTestId('tb-skipped-badge')).toBeInTheDocument();
        });
        expect(screen.getByTestId('tb-skipped-badge')).toHaveAttribute('title', 'TigerBeetle client is not connected');
        expect(screen.getByText('TigerBeetle client is not connected')).toBeInTheDocument();
    });

    it('celebrates when there are zero discrepancies', async () => {
        endpoints.runReconciliation.mockResolvedValue({ data: { data: balancedReport } });
        renderPage();

        await waitFor(() => {
            expect(screen.getByText('Books balanced')).toBeInTheDocument();
        });
        expect(screen.getByText(/agrees with the ledger/i)).toBeInTheDocument();
        expect(screen.getByText('Compared')).toBeInTheDocument();
        expect(screen.getByText('4 accounts · 120 transfers')).toBeInTheDocument();
    });

    it('shows a truncation notice when the discrepancy list is capped', async () => {
        endpoints.runReconciliation.mockResolvedValue({
            data: { data: { ...driftReport, truncated: true, total_discrepancies: 150 } }
        });
        renderPage();

        await waitFor(() => {
            expect(screen.getByText(/Showing the first 2 of 150 discrepancies/)).toBeInTheDocument();
        });
    });

    it('records a run from the Run & record button (the explicit audit action)', async () => {
        endpoints.runReconciliation.mockResolvedValue({ data: { data: balancedReport } });
        renderPage();

        await waitFor(() => {
            expect(screen.getByText('Books balanced')).toBeInTheDocument();
        });
        // The auto-load is the ephemeral GET; the button POSTs to the audit trail.
        fireEvent.click(screen.getByRole('button', { name: /run & record/i }));

        await waitFor(() => {
            expect(endpoints.recordReconciliation).toHaveBeenCalledTimes(1);
        });
    });

    it('shows the run history when there are recorded runs', async () => {
        endpoints.runReconciliation.mockResolvedValue({ data: { data: balancedReport } });
        endpoints.getReconciliationRuns.mockResolvedValue({
            data: {
                data: [
                    { id: 'run-1', run_at: '2026-08-14T10:00:00Z', invoices_checked: 42, total_discrepancies: 0, run_by: null },
                    { id: 'run-2', run_at: '2026-08-13T10:00:00Z', invoices_checked: 40, total_discrepancies: 3, run_by: null },
                ],
            },
        });
        renderPage();

        await waitFor(() => {
            expect(screen.getByText('Run history')).toBeInTheDocument();
        });
        // The drifted historical run shows its invoices-checked count (40 is
        // unique to history — the current report checked 42).
        expect(screen.getByText('40')).toBeInTheDocument();
    });

    it('shows an error state with retry when the run fails', async () => {
        endpoints.runReconciliation.mockRejectedValueOnce(new Error('boom'));
        endpoints.runReconciliation.mockResolvedValueOnce({ data: { data: balancedReport } });
        renderPage();

        await waitFor(() => {
            expect(screen.getByText('Failed to run reconciliation')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByRole('button', { name: /retry/i }));
        await waitFor(() => {
            expect(screen.getByText('Books balanced')).toBeInTheDocument();
        });
    });
});
