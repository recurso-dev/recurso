import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import CreateCustomer from '../CreateCustomer';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { endpoints } from '../../lib/api';
import { ToastProvider } from '../../components/Toast';

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
    endpoints: { createCustomer: vi.fn() },
}));

const renderForm = () =>
    render(
        <MemoryRouter>
            <ToastProvider>
                <CreateCustomer />
            </ToastProvider>
        </MemoryRouter>
    );

describe('CreateCustomer (redesign — Sheet form)', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        endpoints.createCustomer.mockResolvedValue({ data: {} });
    });

    it('renders the slide-over with its title and fields', () => {
        renderForm();
        expect(screen.getByText('Add new customer')).toBeInTheDocument();
        expect(screen.getByLabelText(/Customer name/)).toBeInTheDocument();
        expect(screen.getByLabelText(/Email address/)).toBeInTheDocument();
    });

    it('validates required fields before submitting', async () => {
        renderForm();
        fireEvent.submit(document.getElementById('create-customer-form'));

        await waitFor(() => {
            expect(screen.getByText('Customer name is required.')).toBeInTheDocument();
        });
        expect(endpoints.createCustomer).not.toHaveBeenCalled();
    });

    it('submits the create-customer payload (US default)', async () => {
        renderForm();

        await userEvent.type(screen.getByLabelText(/Customer name/), 'Acme Corporation');
        await userEvent.type(screen.getByLabelText(/Email address/), 'billing@acme.com');

        fireEvent.submit(document.getElementById('create-customer-form'));

        await waitFor(() => {
            expect(endpoints.createCustomer).toHaveBeenCalledTimes(1);
        });
        const payload = endpoints.createCustomer.mock.calls[0][0];
        expect(payload).toMatchObject({
            name: 'Acme Corporation',
            email: 'billing@acme.com',
            country: 'US',
            gstin: '',
            place_of_supply: '',
        });
        expect(navigateMock).toHaveBeenCalledWith('/customers');
    });
});
