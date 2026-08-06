import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import CreateCreditNote from '../CreateCreditNote';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { endpoints } from '../../lib/api';

// jsdom lacks these; Radix (Sheet/Select) touches them.
beforeEach(() => {
    if (!Element.prototype.hasPointerCapture) Element.prototype.hasPointerCapture = () => false;
    if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = () => {};
});

const navigateMock = vi.fn();
vi.mock('react-router', async () => {
    const actual = await vi.importActual('react-router');
    return { ...actual, useNavigate: () => navigateMock };
});

vi.mock('../../lib/api', () => ({
    endpoints: {
        getCustomers: vi.fn(),
        getInvoices: vi.fn(),
        createCreditNote: vi.fn(),
    },
}));

const ACME = { id: 'c0000000-0000-0000-0000-000000000001', name: 'Acme Corp', email: 'ops@acme.com' };
const GLOBEX = { id: 'c0000000-0000-0000-0000-000000000002', name: 'Globex', email: 'ap@globex.com' };

// One INR invoice for Acme, one USD invoice for Globex — proves the picker is
// scoped to the selected customer and that the currency follows the invoice.
const ACME_INV = {
    id: 'i0000000-0000-0000-0000-0000000000a1',
    invoice_number: 'INV-ACME-01',
    customer_id: ACME.id,
    total: 500000,
    currency: 'INR',
    status: 'open',
};
const GLOBEX_INV = {
    id: 'i0000000-0000-0000-0000-0000000000b1',
    invoice_number: 'INV-GLBX-01',
    customer_id: GLOBEX.id,
    total: 4200,
    currency: 'USD',
    status: 'paid',
};

const renderPage = () =>
    render(
        <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
            <MemoryRouter>
                <CreateCreditNote />
            </MemoryRouter>
        </QueryClientProvider>
    );

const pickOption = async (user, triggerId, optionText) => {
    await user.click(document.getElementById(triggerId));
    const option = await screen.findByRole('option', { name: optionText });
    await user.click(option);
};

describe('CreateCreditNote — linked-invoice picker', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        endpoints.getCustomers.mockResolvedValue({ data: { data: [ACME, GLOBEX] } });
        endpoints.getInvoices.mockResolvedValue({ data: { data: [ACME_INV, GLOBEX_INV] } });
        endpoints.createCreditNote.mockResolvedValue({ data: { data: { id: 'cn-1' } } });
    });

    it('disables the invoice picker until a customer is chosen', async () => {
        renderPage();
        await waitFor(() => expect(endpoints.getCustomers).toHaveBeenCalled());
        expect(screen.getByText('Select a customer first')).toBeInTheDocument();
        expect(document.getElementById('invoice_id')).toBeDisabled();
    });

    it('only offers the selected customer\'s invoices and links the real invoice id (never a typed UUID)', async () => {
        const user = userEvent.setup();
        renderPage();
        await waitFor(() => expect(endpoints.getCustomers).toHaveBeenCalled());

        await pickOption(user, 'customer_id', /Acme Corp/);
        // Open the now-enabled invoice picker.
        await user.click(document.getElementById('invoice_id'));

        // Acme's invoice is offered; Globex's is not (scoped to the customer).
        expect(await screen.findByRole('option', { name: /INV-ACME-01/ })).toBeInTheDocument();
        expect(screen.queryByRole('option', { name: /INV-GLBX-01/ })).not.toBeInTheDocument();

        await user.click(await screen.findByRole('option', { name: /INV-ACME-01/ }));

        // Linking the invoice adopts its currency (INR) for the amount prefix.
        await waitFor(() => expect(screen.getByText('INR')).toBeInTheDocument());

        fireEvent.change(document.getElementById('amount'), { target: { value: '1000' } });
        fireEvent.submit(document.getElementById('create-credit-note-form'));

        await waitFor(() => expect(endpoints.createCreditNote).toHaveBeenCalledTimes(1));
        const payload = endpoints.createCreditNote.mock.calls[0][0];
        expect(payload.invoice_id).toBe(ACME_INV.id);
        expect(payload.currency).toBe('INR');
    });

    it('resets the linked invoice when the customer changes', async () => {
        const user = userEvent.setup();
        renderPage();
        await waitFor(() => expect(endpoints.getCustomers).toHaveBeenCalled());

        await pickOption(user, 'customer_id', /Acme Corp/);
        await pickOption(user, 'invoice_id', /INV-ACME-01/);
        await waitFor(() => expect(screen.getByText('INR')).toBeInTheDocument());

        // Switch to Globex — the Acme link and its INR currency must clear:
        // the amount prefix returns to the USD default.
        await pickOption(user, 'customer_id', /Globex/);
        await waitFor(() => expect(screen.getByText('USD')).toBeInTheDocument());
        expect(screen.queryByText('INR')).not.toBeInTheDocument();

        // ...and a submit now carries no linked invoice.
        fireEvent.change(document.getElementById('amount'), { target: { value: '25' } });
        fireEvent.submit(document.getElementById('create-credit-note-form'));
        await waitFor(() => expect(endpoints.createCreditNote).toHaveBeenCalledTimes(1));
        expect(endpoints.createCreditNote.mock.calls[0][0].invoice_id).toBeUndefined();
    });
});
