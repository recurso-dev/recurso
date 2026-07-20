import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import PlanCharges from '../PlanCharges';
import { ToastProvider } from '../../Toast';
import { endpoints } from '../../../lib/api';

vi.mock('../../../lib/api', () => ({
    endpoints: {
        getPlanCharges: vi.fn(),
        setPlanCharges: vi.fn(),
        getBillableMetrics: vi.fn(),
    },
}));

const metrics = [
    { id: 'metric-1', code: 'api_calls', name: 'API Calls' },
    { id: 'metric-2', code: 'storage', name: 'Storage GB' },
];

const renderCharges = () =>
    render(
        <ToastProvider>
            <PlanCharges planId="plan-123" currency="USD" />
        </ToastProvider>
    );

describe('PlanCharges editor', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        endpoints.getPlanCharges.mockResolvedValue({ data: { data: [] } });
        endpoints.getBillableMetrics.mockResolvedValue({ data: { data: metrics } });
        endpoints.setPlanCharges.mockResolvedValue({ data: { data: [] } });
    });

    it('renders existing charges read-only with a pricing summary', async () => {
        endpoints.getPlanCharges.mockResolvedValue({
            data: {
                data: [
                    {
                        id: 'ch-1',
                        metric_id: 'metric-1',
                        charge_model: 'package',
                        amounts: { USD: { package_amount: 500, package_size: 1000 } },
                        metric: { name: 'API Calls' },
                    },
                ],
            },
        });
        renderCharges();
        expect(await screen.findByText('API Calls')).toBeInTheDocument();
        expect(screen.getByText('$5.00 per 1000 units')).toBeInTheDocument();
    });

    it('saves a per-unit charge with the plan currency and metric id', async () => {
        renderCharges();
        fireEvent.click(await screen.findByRole('button', { name: /edit/i }));
        fireEvent.click(screen.getByRole('button', { name: /add charge/i }));

        fireEvent.change(screen.getByLabelText('Metric 1'), { target: { value: 'metric-1' } });
        fireEvent.change(screen.getByLabelText('Per-unit rate 1'), { target: { value: '0.0035' } });
        fireEvent.click(screen.getByRole('button', { name: /save charges/i }));

        await waitFor(() =>
            expect(endpoints.setPlanCharges).toHaveBeenCalledWith('plan-123', [
                {
                    metric_id: 'metric-1',
                    charge_model: 'per_unit',
                    amounts: { USD: { unit_amount: '0.0035' } },
                    hsn_code: '',
                },
            ])
        );
    });

    it('converts graduated tiers to minor-unit flat fees and a null last bound', async () => {
        renderCharges();
        fireEvent.click(await screen.findByRole('button', { name: /edit/i }));
        fireEvent.click(screen.getByRole('button', { name: /add charge/i }));

        fireEvent.change(screen.getByLabelText('Metric 1'), { target: { value: 'metric-2' } });
        fireEvent.change(screen.getByLabelText('Charge model 1'), { target: { value: 'graduated' } });
        // one tier exists (unbounded). Add a second so the first becomes bounded.
        fireEvent.click(screen.getByRole('button', { name: /add tier/i }));

        fireEvent.change(screen.getByLabelText('Charge 1 tier 1 up to'), { target: { value: '100' } });
        fireEvent.change(screen.getByLabelText('Charge 1 tier 1 rate'), { target: { value: '1.00' } });
        fireEvent.change(screen.getByLabelText('Charge 1 tier 1 flat fee'), { target: { value: '2.50' } });
        fireEvent.change(screen.getByLabelText('Charge 1 tier 2 rate'), { target: { value: '0.50' } });

        fireEvent.click(screen.getByRole('button', { name: /save charges/i }));

        await waitFor(() =>
            expect(endpoints.setPlanCharges).toHaveBeenCalledWith('plan-123', [
                {
                    metric_id: 'metric-2',
                    charge_model: 'graduated',
                    amounts: {
                        USD: {
                            tiers: [
                                { up_to: 100, unit_amount: '1.00', flat_amount: 250 },
                                { up_to: null, unit_amount: '0.50', flat_amount: 0 },
                            ],
                        },
                    },
                    hsn_code: '',
                },
            ])
        );
    });

    it('blocks saving when a tier bound does not increase', async () => {
        renderCharges();
        fireEvent.click(await screen.findByRole('button', { name: /edit/i }));
        fireEvent.click(screen.getByRole('button', { name: /add charge/i }));
        fireEvent.change(screen.getByLabelText('Metric 1'), { target: { value: 'metric-1' } });
        fireEvent.change(screen.getByLabelText('Charge model 1'), { target: { value: 'volume' } });
        fireEvent.click(screen.getByRole('button', { name: /add tier/i }));
        // leave tier 1 up_to empty -> invalid whole number
        fireEvent.change(screen.getByLabelText('Charge 1 tier 1 rate'), { target: { value: '1.00' } });
        fireEvent.change(screen.getByLabelText('Charge 1 tier 2 rate'), { target: { value: '0.50' } });
        fireEvent.click(screen.getByRole('button', { name: /save charges/i }));

        expect(await screen.findByRole('alert')).toBeInTheDocument();
        expect(endpoints.setPlanCharges).not.toHaveBeenCalled();
    });
});
