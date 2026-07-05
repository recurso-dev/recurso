import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, it, expect } from 'vitest';
import SetupChecklist from '../SetupChecklist';

const renderChecklist = (props = {}) =>
    render(
        <MemoryRouter>
            <SetupChecklist
                plans={[]}
                customers={[]}
                subscriptions={[]}
                invoices={[]}
                {...props}
            />
        </MemoryRouter>
    );

describe('SetupChecklist', () => {
    it('renders all 4 steps when nothing is set up', () => {
        renderChecklist();

        expect(screen.getByText('Get set up')).toBeInTheDocument();
        expect(screen.getByText('Create a plan')).toBeInTheDocument();
        expect(screen.getByText('Add a customer')).toBeInTheDocument();
        expect(screen.getByText('Start a subscription')).toBeInTheDocument();
        expect(screen.getByText('See your first invoice')).toBeInTheDocument();

        expect(screen.getByText('0 of 4 complete')).toBeInTheDocument();

        // Every step is an incomplete link pointing at the right page
        expect(screen.getByRole('link', { name: /create a plan/i })).toHaveAttribute('href', '/plans');
        expect(screen.getByRole('link', { name: /add a customer/i })).toHaveAttribute('href', '/customers');
        expect(screen.getByRole('link', { name: /start a subscription/i })).toHaveAttribute('href', '/subscriptions');
        expect(screen.getByRole('link', { name: /see your first invoice/i })).toHaveAttribute('href', '/invoices');
    });

    it('marks done states correctly from props', () => {
        renderChecklist({
            plans: [{ id: 'plan_1' }],
            customers: [{ id: 'cus_1' }],
            subscriptions: [],
            invoices: [],
        });

        expect(screen.getByText('2 of 4 complete')).toBeInTheDocument();

        // Completed steps are no longer links
        expect(screen.queryByRole('link', { name: /create a plan/i })).not.toBeInTheDocument();
        expect(screen.queryByRole('link', { name: /add a customer/i })).not.toBeInTheDocument();

        // ...but still rendered as completed rows
        expect(screen.getByText('Create a plan')).toBeInTheDocument();
        expect(screen.getByText('Add a customer')).toBeInTheDocument();

        // Incomplete steps remain links
        expect(screen.getByRole('link', { name: /start a subscription/i })).toHaveAttribute('href', '/subscriptions');
        expect(screen.getByRole('link', { name: /see your first invoice/i })).toHaveAttribute('href', '/invoices');

        // Progress bar reflects completion
        const progress = screen.getByRole('progressbar');
        expect(progress).toHaveAttribute('aria-valuenow', '2');
        expect(progress).toHaveAttribute('aria-valuemax', '4');
    });

    it('accepts numeric counts as props', () => {
        renderChecklist({ plans: 3, customers: 0, subscriptions: 1, invoices: 0 });

        expect(screen.getByText('2 of 4 complete')).toBeInTheDocument();
        expect(screen.queryByRole('link', { name: /create a plan/i })).not.toBeInTheDocument();
        expect(screen.getByRole('link', { name: /add a customer/i })).toBeInTheDocument();
    });

    it('renders nothing when all steps are complete', () => {
        const { container } = renderChecklist({
            plans: [{ id: 'plan_1' }],
            customers: [{ id: 'cus_1' }],
            subscriptions: [{ id: 'sub_1' }],
            invoices: [{ id: 'inv_1' }],
        });

        expect(container).toBeEmptyDOMElement();
        expect(screen.queryByText('Get set up')).not.toBeInTheDocument();
    });
});
