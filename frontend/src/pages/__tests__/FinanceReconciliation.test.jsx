import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import FinanceReconciliation from '../FinanceReconciliation';
import { endpoints } from '../../lib/api';

// Mock the API module
vi.mock('../../lib/api', () => ({
    endpoints: {
        runReconciliation: vi.fn(),
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

const renderPage = () => render(<MemoryRouter><FinanceReconciliation /></MemoryRouter>);

describe('FinanceReconciliation page', () => {
    beforeEach(() => {
        vi.clearAllMocks();
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
        expect(screen.getByText('aaaaaaaa…')).toBeInTheDocument();
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

    it('re-runs reconciliation from the Run again button', async () => {
        endpoints.runReconciliation.mockResolvedValue({ data: { data: balancedReport } });
        renderPage();

        await waitFor(() => {
            expect(screen.getByText('Books balanced')).toBeInTheDocument();
        });
        fireEvent.click(screen.getByRole('button', { name: /run again/i }));

        await waitFor(() => {
            expect(endpoints.runReconciliation).toHaveBeenCalledTimes(2);
        });
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
