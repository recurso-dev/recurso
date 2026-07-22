import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import PlanCharges from '../PlanCharges';
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
        <>
            <PlanCharges planId="plan-123" currency="USD" />
        </>
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
                    pay_in_advance: false,
                    hsn_code: '',
                },
            ])
        );
    });

    it('sends pay_in_advance when checked for an eligible model', async () => {
        renderCharges();
        fireEvent.click(await screen.findByRole('button', { name: /edit/i }));
        fireEvent.click(screen.getByRole('button', { name: /add charge/i }));

        fireEvent.change(screen.getByLabelText('Metric 1'), { target: { value: 'metric-1' } });
        fireEvent.change(screen.getByLabelText('Per-unit rate 1'), { target: { value: '0.0035' } });
        fireEvent.click(screen.getByLabelText('Charge 1 bill in advance'));
        fireEvent.click(screen.getByRole('button', { name: /save charges/i }));

        await waitFor(() =>
            expect(endpoints.setPlanCharges).toHaveBeenCalledWith('plan-123', [
                {
                    metric_id: 'metric-1',
                    charge_model: 'per_unit',
                    amounts: { USD: { unit_amount: '0.0035' } },
                    pay_in_advance: true,
                    hsn_code: '',
                },
            ])
        );
    });

    it('hides the bill-in-advance checkbox for cumulative models', async () => {
        renderCharges();
        fireEvent.click(await screen.findByRole('button', { name: /edit/i }));
        fireEvent.click(screen.getByRole('button', { name: /add charge/i }));
        fireEvent.change(screen.getByLabelText('Metric 1'), { target: { value: 'metric-1' } });
        fireEvent.change(screen.getByLabelText('Charge model 1'), { target: { value: 'graduated' } });

        expect(screen.queryByLabelText('Charge 1 bill in advance')).not.toBeInTheDocument();
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
                    pay_in_advance: false,
                    hsn_code: '',
                },
            ])
        );
    });

    it('saves a percentage charge with money fields scaled to minor units', async () => {
        renderCharges();
        fireEvent.click(await screen.findByRole('button', { name: /edit/i }));
        fireEvent.click(screen.getByRole('button', { name: /add charge/i }));

        fireEvent.change(screen.getByLabelText('Metric 1'), { target: { value: 'metric-1' } });
        fireEvent.change(screen.getByLabelText('Charge model 1'), { target: { value: 'percentage' } });
        fireEvent.change(screen.getByLabelText('Percentage rate 1'), { target: { value: '2.5' } });
        fireEvent.change(screen.getByLabelText('Charge 1 Fixed fee'), { target: { value: '0.30' } });
        fireEvent.change(screen.getByLabelText('Charge 1 Maximum'), { target: { value: '50' } });
        fireEvent.click(screen.getByRole('button', { name: /save charges/i }));

        await waitFor(() =>
            expect(endpoints.setPlanCharges).toHaveBeenCalledWith('plan-123', [
                {
                    metric_id: 'metric-1',
                    charge_model: 'percentage',
                    amounts: {
                        USD: { rate: '2.5', fixed_amount: 30, free_units: 0, min_amount: 0, max_amount: 5000 },
                    },
                    pay_in_advance: false,
                    hsn_code: '',
                },
            ])
        );
    });

    it('scales graduated_percentage tier bounds (money) to minor units', async () => {
        renderCharges();
        fireEvent.click(await screen.findByRole('button', { name: /edit/i }));
        fireEvent.click(screen.getByRole('button', { name: /add charge/i }));

        fireEvent.change(screen.getByLabelText('Metric 1'), { target: { value: 'metric-1' } });
        fireEvent.change(screen.getByLabelText('Charge model 1'), { target: { value: 'graduated_percentage' } });
        fireEvent.click(screen.getByRole('button', { name: /add tier/i }));

        fireEvent.change(screen.getByLabelText('Charge 1 tier 1 up to'), { target: { value: '100' } }); // $100 -> 10000
        fireEvent.change(screen.getByLabelText('Charge 1 tier 1 rate'), { target: { value: '3' } });
        fireEvent.change(screen.getByLabelText('Charge 1 tier 2 rate'), { target: { value: '2' } });
        fireEvent.click(screen.getByRole('button', { name: /save charges/i }));

        await waitFor(() =>
            expect(endpoints.setPlanCharges).toHaveBeenCalledWith('plan-123', [
                {
                    metric_id: 'metric-1',
                    charge_model: 'graduated_percentage',
                    amounts: {
                        USD: {
                            tiers: [
                                { up_to: 10000, flat_amount: 0, rate: '3' },
                                { up_to: null, flat_amount: 0, rate: '2' },
                            ],
                        },
                    },
                    pay_in_advance: false,
                    hsn_code: '',
                },
            ])
        );
    });

    it('saves a dynamic charge with an empty amounts entry (priced per event)', async () => {
        renderCharges();
        fireEvent.click(await screen.findByRole('button', { name: /edit/i }));
        fireEvent.click(screen.getByRole('button', { name: /add charge/i }));

        fireEvent.change(screen.getByLabelText('Metric 1'), { target: { value: 'metric-2' } });
        fireEvent.change(screen.getByLabelText('Charge model 1'), { target: { value: 'dynamic' } });
        fireEvent.click(screen.getByRole('button', { name: /save charges/i }));

        await waitFor(() =>
            expect(endpoints.setPlanCharges).toHaveBeenCalledWith('plan-123', [
                {
                    metric_id: 'metric-2',
                    charge_model: 'dynamic',
                    amounts: { USD: {} },
                    pay_in_advance: false,
                    hsn_code: '',
                },
            ])
        );
    });

    it('summarizes a percentage charge read-only', async () => {
        endpoints.getPlanCharges.mockResolvedValue({
            data: {
                data: [
                    {
                        id: 'ch-p',
                        metric_id: 'metric-1',
                        charge_model: 'percentage',
                        amounts: { USD: { rate: '2.5', fixed_amount: 30 } },
                        metric: { name: 'API Calls' },
                    },
                ],
            },
        });
        renderCharges();
        expect(await screen.findByText('2.5% of value + $0.30 fee')).toBeInTheDocument();
    });

    it('saves per-value filter pricing for a per_unit charge (A4)', async () => {
        renderCharges();
        fireEvent.click(await screen.findByRole('button', { name: /edit/i }));
        fireEvent.click(screen.getByRole('button', { name: /add charge/i }));

        fireEvent.change(screen.getByLabelText('Metric 1'), { target: { value: 'metric-1' } });
        fireEvent.change(screen.getByLabelText('Per-unit rate 1'), { target: { value: '0.01' } });
        fireEvent.change(screen.getByLabelText('Charge 1 filter property'), { target: { value: 'region' } });
        fireEvent.click(screen.getByRole('button', { name: /add value/i }));
        fireEvent.change(screen.getByLabelText('Charge 1 filter 1 value'), { target: { value: 'us' } });
        fireEvent.change(screen.getByLabelText('Charge 1 filter 1 rate'), { target: { value: '0.02' } });
        fireEvent.click(screen.getByRole('button', { name: /save charges/i }));

        await waitFor(() =>
            expect(endpoints.setPlanCharges).toHaveBeenCalledWith('plan-123', [
                {
                    metric_id: 'metric-1',
                    charge_model: 'per_unit',
                    amounts: { USD: { unit_amount: '0.01' } },
                    hsn_code: '',
                    pay_in_advance: false,
                    filter_key: 'region',
                    filters: [{ value: 'us', amounts: { USD: { unit_amount: '0.02' } } }],
                },
            ])
        );
    });

    it('hides the filter builder for tier/dynamic models', async () => {
        renderCharges();
        fireEvent.click(await screen.findByRole('button', { name: /edit/i }));
        fireEvent.click(screen.getByRole('button', { name: /add charge/i }));
        fireEvent.change(screen.getByLabelText('Metric 1'), { target: { value: 'metric-1' } });
        fireEvent.change(screen.getByLabelText('Charge model 1'), { target: { value: 'graduated' } });

        expect(screen.queryByLabelText('Charge 1 filter property')).not.toBeInTheDocument();
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
