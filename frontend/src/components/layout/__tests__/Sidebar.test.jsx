import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router';
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
        expect(screen.getByText('Billing')).toBeInTheDocument();
        expect(screen.getByText('Usage')).toBeInTheDocument();
        expect(screen.getByText('Payments')).toBeInTheDocument();
        expect(screen.getByText('Books')).toBeInTheDocument();
        expect(screen.getByText('Reports')).toBeInTheDocument();
        expect(screen.getByText('Revenue Recovery')).toBeInTheDocument();
        expect(screen.getByText('System')).toBeInTheDocument();
    });

    it('renders the main navigation links', () => {
        renderWithRouter(<Sidebar />);
        ['Home', 'Customers', 'Plans', 'Subscriptions', 'Invoices',
            'Coupons', 'Referrals', 'Gifts', 'Dunning',
            'Ledger', 'Reconciliation', 'Usage Explorer',
            'Developers', 'Settings'].forEach((label) => {
            expect(screen.getByText(label)).toBeInTheDocument();
        });
    });

    it('links to the documentation site', () => {
        renderWithRouter(<Sidebar />);
        const docsLink = screen.getByText('Documentation').closest('a');
        expect(docsLink).toHaveAttribute('href', 'https://docs.recurso.dev');
        expect(docsLink).toHaveAttribute('target', '_blank');
    });

    it('applies the primary-token active style to the current route', () => {
        renderWithRouter(<Sidebar />, { route: '/customers' });

        const customersLink = screen.getByText('Customers').closest('a');
        expect(customersLink).toHaveClass('bg-primary/10');
        expect(customersLink).toHaveClass('text-primary');

        // Home uses exact matching, so it must NOT be active on /customers.
        const homeLink = screen.getByText('Home').closest('a');
        expect(homeLink).not.toHaveClass('bg-primary/10');
    });

    it('scales the active indicator bar in on the current route only', () => {
        renderWithRouter(<Sidebar />, { route: '/customers' });

        // The indicator is the aria-hidden bar inside each nav link.
        const barOf = (label) =>
            screen.getByText(label).closest('a').querySelector('[aria-hidden="true"]');

        expect(barOf('Customers')).toHaveClass('scale-y-100');
        expect(barOf('Customers')).not.toHaveClass('scale-y-0');
        // Transform-only transition — no layout animation.
        expect(barOf('Customers')).toHaveClass('transition-transform');

        expect(barOf('Home')).toHaveClass('scale-y-0');
    });
});
