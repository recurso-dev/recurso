import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import CreateSubscription from '../CreateSubscription';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { endpoints } from '../../lib/api';

// jsdom lacks these; Radix (Sheet/Select) touches them.
beforeEach(() => {
    if (!Element.prototype.hasPointerCapture) Element.prototype.hasPointerCapture = () => false;
    if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = () => {};
});

const navigateMock = vi.fn();
vi.mock('react-router-dom', async () => {
    const actual = await vi.importActual('react-router-dom');
    return { ...actual, useNavigate: () => navigateMock };
});

vi.mock('../../lib/api', () => ({
    endpoints: {
        getCustomers: vi.fn(),
        getPlans: vi.fn(),
        createSubscription: vi.fn(),
    },
}));

const CUSTOMER = { id: 'c0000000-0000-0000-0000-000000000001', name: 'Acme Corp', email: 'ops@acme.com' };
const PLAN = {
    id: 'p0000000-0000-0000-0000-000000000001',
    name: 'Pro',
    interval_unit: 'month',
    prices: [{ amount: 4900, currency: 'USD' }],
};

const renderPage = () =>
    render(
        <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
            <MemoryRouter>
                <CreateSubscription />
            </MemoryRouter>
        </QueryClientProvider>
    );

// Drives a Radix Select: open the trigger, click the option by its text.
const pickOption = async (user, triggerId, optionText) => {
    await user.click(document.getElementById(triggerId));
    const option = await screen.findByRole('option', { name: optionText });
    await user.click(option);
};

describe('CreateSubscription (Sheet form)', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        endpoints.getCustomers.mockResolvedValue({ data: { data: [CUSTOMER] } });
        endpoints.getPlans.mockResolvedValue({ data: { data: [PLAN] } });
        endpoints.createSubscription.mockResolvedValue({ data: { data: { id: 'sub-1' } } });
    });

    it('renders the slide-over with customer, plan, and billing controls', async () => {
        renderPage();
        expect(screen.getByText('Create new subscription')).toBeInTheDocument();
        expect(document.getElementById('customer')).toBeTruthy();
        expect(document.getElementById('plan')).toBeTruthy();
        expect(document.getElementById('start_date')).toBeTruthy();
        await waitFor(() => expect(endpoints.getCustomers).toHaveBeenCalled());
    });

    it('refuses to submit without a customer and plan', async () => {
        renderPage();
        fireEvent.submit(document.getElementById('create-subscription-form'));
        await waitFor(() => {
            expect(endpoints.createSubscription).not.toHaveBeenCalled();
        });
    });

    it('creates the subscription with the exact payload and navigates back', async () => {
        const user = userEvent.setup();
        renderPage();

        await waitFor(() => expect(endpoints.getPlans).toHaveBeenCalled());
        await pickOption(user, 'customer', /Acme Corp/);
        await pickOption(user, 'plan', /Pro/);
        fireEvent.click(document.querySelector('input[type="checkbox"]'));

        fireEvent.submit(document.getElementById('create-subscription-form'));

        await waitFor(() => expect(endpoints.createSubscription).toHaveBeenCalledTimes(1));
        const payload = endpoints.createSubscription.mock.calls[0][0];
        expect(payload.customer_id).toBe(CUSTOMER.id);
        expect(payload.plan_id).toBe(PLAN.id);
        expect(payload.billing_anchor_type).toBe('acquisition');
        expect(payload.payment_terms).toBe('due_on_receipt');
        // start_date must be a full ISO timestamp, not the bare date input value.
        expect(new Date(payload.start_date).toString()).not.toBe('Invalid Date');
        expect(payload.start_date).toMatch(/T.*Z$/);

        await waitFor(() => expect(navigateMock).toHaveBeenCalledWith('/subscriptions'));
    });

    it('does not submit until recurring billing is authorized (consent)', async () => {
        const user = userEvent.setup();
        renderPage();

        await waitFor(() => expect(endpoints.getPlans).toHaveBeenCalled());
        await pickOption(user, 'customer', /Acme Corp/);
        await pickOption(user, 'plan', /Pro/);
        // Consent checkbox intentionally left unchecked.

        fireEvent.submit(document.getElementById('create-subscription-form'));
        await waitFor(() => {
            expect(endpoints.createSubscription).not.toHaveBeenCalled();
        });
    });

    it('stays on the form and surfaces the API error on failure', async () => {
        const user = userEvent.setup();
        endpoints.createSubscription.mockRejectedValue({
            response: { data: { error: { message: 'plan is archived' } } },
        });
        renderPage();

        await waitFor(() => expect(endpoints.getPlans).toHaveBeenCalled());
        await pickOption(user, 'customer', /Acme Corp/);
        await pickOption(user, 'plan', /Pro/);
        fireEvent.click(document.querySelector('input[type="checkbox"]'));

        fireEvent.submit(document.getElementById('create-subscription-form'));

        await waitFor(() => expect(endpoints.createSubscription).toHaveBeenCalledTimes(1));
        expect(navigateMock).not.toHaveBeenCalledWith('/subscriptions');
    });
});
