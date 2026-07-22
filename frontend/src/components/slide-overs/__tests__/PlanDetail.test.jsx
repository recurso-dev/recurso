import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import PlanDetail from '../PlanDetail';
import { endpoints } from '../../../lib/api';

// Mock the API module
vi.mock('../../../lib/api', () => ({
    endpoints: {
        getPlanEntitlements: vi.fn(),
        setPlanEntitlements: vi.fn(),
        getPlanCharges: vi.fn(),
        setPlanCharges: vi.fn(),
        getBillableMetrics: vi.fn(),
    }
}));

const plan = {
    id: 'plan-123',
    name: 'Pro Tier',
    code: 'pro-monthly',
    active: true,
    interval_unit: 'month',
    interval_count: 1,
    created_at: '2026-01-01T00:00:00Z',
    prices: [{ amount: 9900, currency: 'usd' }],
};

const existingEntitlements = [
    { feature_key: 'api.access', kind: 'boolean', bool_value: true },
    { feature_key: 'seats', kind: 'limit', limit_value: 10 },
];

const renderPlanDetail = () => render(
    <>
        <PlanDetail plan={plan} isOpen={true} onClose={() => { }} />
    </>
);

describe('PlanDetail entitlements editor', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        endpoints.getPlanEntitlements.mockResolvedValue({ data: { data: existingEntitlements } });
        endpoints.setPlanEntitlements.mockResolvedValue({ data: { data: existingEntitlements } });
        endpoints.getPlanCharges.mockResolvedValue({ data: { data: [] } });
        endpoints.getBillableMetrics.mockResolvedValue({ data: { data: [] } });
        endpoints.setPlanCharges.mockResolvedValue({ data: { data: [] } });
    });

    it('renders the fetched entitlements read-only', async () => {
        renderPlanDetail();

        await waitFor(() => {
            expect(screen.getByText('api.access')).toBeInTheDocument();
        });
        expect(screen.getByText('seats')).toBeInTheDocument();
        expect(screen.getByText('limit: 10')).toBeInTheDocument();
        expect(endpoints.getPlanEntitlements).toHaveBeenCalledWith('plan-123');
    });

    it('adds a row and saves the full replacement set (PUT semantics)', async () => {
        renderPlanDetail();
        await waitFor(() => expect(screen.getByText('api.access')).toBeInTheDocument());

        fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
        fireEvent.click(screen.getByRole('button', { name: /add entitlement/i }));

        // The new row is the third one
        fireEvent.change(screen.getByLabelText('Feature key 3'), { target: { value: 'exports.csv' } });
        fireEvent.change(screen.getByLabelText('Kind 3'), { target: { value: 'limit' } });
        fireEvent.change(screen.getByLabelText('Limit value 3'), { target: { value: '500' } });

        fireEvent.click(screen.getByRole('button', { name: /save entitlements/i }));

        await waitFor(() => {
            expect(endpoints.setPlanEntitlements).toHaveBeenCalledWith('plan-123', [
                { feature_key: 'api.access', kind: 'boolean', bool_value: true },
                { feature_key: 'seats', kind: 'limit', limit_value: 10 },
                { feature_key: 'exports.csv', kind: 'limit', limit_value: 500 },
            ]);
        });
    });

    it('removes a row and saves without it (absent keys are removed)', async () => {
        renderPlanDetail();
        await waitFor(() => expect(screen.getByText('api.access')).toBeInTheDocument());

        fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
        fireEvent.click(screen.getByLabelText('Remove entitlement 1'));
        fireEvent.click(screen.getByRole('button', { name: /save entitlements/i }));

        await waitFor(() => {
            expect(endpoints.setPlanEntitlements).toHaveBeenCalledWith('plan-123', [
                { feature_key: 'seats', kind: 'limit', limit_value: 10 },
            ]);
        });
    });

    it('blocks saving on an invalid feature key and does not call the API', async () => {
        renderPlanDetail();
        await waitFor(() => expect(screen.getByText('api.access')).toBeInTheDocument());

        fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
        fireEvent.click(screen.getByRole('button', { name: /add entitlement/i }));
        fireEvent.change(screen.getByLabelText('Feature key 3'), { target: { value: '!!bad key' } });
        fireEvent.click(screen.getByRole('button', { name: /save entitlements/i }));

        expect(await screen.findByRole('alert')).toHaveTextContent(/may only contain/i);
        expect(endpoints.setPlanEntitlements).not.toHaveBeenCalled();
    });

    it('rejects duplicate feature keys client-side', async () => {
        renderPlanDetail();
        await waitFor(() => expect(screen.getByText('api.access')).toBeInTheDocument());

        fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
        fireEvent.click(screen.getByRole('button', { name: /add entitlement/i }));
        fireEvent.change(screen.getByLabelText('Feature key 3'), { target: { value: 'seats' } });
        fireEvent.click(screen.getByRole('button', { name: /save entitlements/i }));

        expect(await screen.findByRole('alert')).toHaveTextContent(/duplicate feature key/i);
        expect(endpoints.setPlanEntitlements).not.toHaveBeenCalled();
    });
});
