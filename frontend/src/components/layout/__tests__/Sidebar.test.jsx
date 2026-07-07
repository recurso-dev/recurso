import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import Sidebar from '../Sidebar';
import { describe, it, expect } from 'vitest';

const renderWithRouter = (ui, { route = '/' } = {}) => {
    window.history.pushState({}, 'Test page', route);
    return render(ui, { wrapper: BrowserRouter });
};

describe('Sidebar (redesign)', () => {
    it('renders the brand name', () => {
        renderWithRouter(<Sidebar />);
        expect(screen.getByText('Recurso')).toBeInTheDocument();
    });

    it('renders the grouped navigation sections', () => {
        renderWithRouter(<Sidebar />);
        expect(screen.getByText('Core')).toBeInTheDocument();
        expect(screen.getByText('Growth')).toBeInTheDocument();
        expect(screen.getByText('Finance')).toBeInTheDocument();
        expect(screen.getByText('System')).toBeInTheDocument();
    });

    it('renders the main navigation links', () => {
        renderWithRouter(<Sidebar />);
        ['Home', 'Customers', 'Plans', 'Subscriptions', 'Invoices',
            'Coupons', 'Referrals', 'Gifts', 'Dunning',
            'Ledger', 'Reconciliation', 'Usage',
            'Developers', 'Settings'].forEach((label) => {
            expect(screen.getByText(label)).toBeInTheDocument();
        });
    });

    it('applies the emerald active style to the current route', () => {
        renderWithRouter(<Sidebar />, { route: '/customers' });

        const customersLink = screen.getByText('Customers').closest('a');
        expect(customersLink).toHaveClass('bg-emerald-50');
        expect(customersLink).toHaveClass('text-emerald-700');

        // Home uses exact matching, so it must NOT be active on /customers.
        const homeLink = screen.getByText('Home').closest('a');
        expect(homeLink).not.toHaveClass('bg-emerald-50');
    });
});
