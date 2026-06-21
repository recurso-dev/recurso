import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import Sidebar from '../Sidebar';
import { describe, it, expect, vi } from 'vitest';

// Mock the Icon component since it might import SVGs or other assets
vi.mock('../../ui/Icon', () => ({
    default: ({ name, className }) => <span data-testid={`icon-${name}`} className={className}></span>
}));

const renderWithRouter = (ui, { route = '/' } = {}) => {
    window.history.pushState({}, 'Test page', route);
    return render(ui, { wrapper: BrowserRouter });
};

describe('Sidebar Component', () => {
    it('renders the brand name correctly', () => {
        renderWithRouter(<Sidebar />);
        expect(screen.getByText('Recurso')).toBeInTheDocument();
    });

    it('renders navigation sections', () => {
        renderWithRouter(<Sidebar />);
        expect(screen.getByText('Core')).toBeInTheDocument();
        expect(screen.getByText('System')).toBeInTheDocument();
    });

    it('renders all main navigation links', () => {
        renderWithRouter(<Sidebar />);
        expect(screen.getByText('Home')).toBeInTheDocument();
        expect(screen.getByText('Customers')).toBeInTheDocument();
        expect(screen.getByText('Plans')).toBeInTheDocument();
        expect(screen.getByText('Products')).toBeInTheDocument();
        expect(screen.getByText('Coupons')).toBeInTheDocument();
        expect(screen.getByText('Subscriptions')).toBeInTheDocument();
        expect(screen.getByText('Invoices')).toBeInTheDocument();
        expect(screen.getByText('Credit Notes')).toBeInTheDocument();
        expect(screen.getByText('Financials')).toBeInTheDocument();
        expect(screen.getByText('Usage')).toBeInTheDocument();
        expect(screen.getByText('Settings')).toBeInTheDocument();
    });

    it('applies active styles to the current route', () => {
        renderWithRouter(<Sidebar />, { route: '/customers' });

        const customersLink = screen.getByText('Customers').closest('a');
        expect(customersLink).toHaveClass('bg-gray-100');
        expect(customersLink).toHaveClass('text-gray-900');
        expect(customersLink).toHaveClass('font-semibold');

        const homeLink = screen.getByText('Home').closest('a');
        expect(homeLink).not.toHaveClass('bg-gray-100');
        expect(homeLink).toHaveClass('text-gray-500');
    });

    it('applies hover styles to inactive links (implicit check via class presence)', () => {
        renderWithRouter(<Sidebar />, { route: '/' });
        const customersLink = screen.getByText('Customers').closest('a');
        expect(customersLink).toHaveClass('hover:bg-gray-50');
    });
});
