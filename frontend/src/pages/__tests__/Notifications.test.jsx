import { render, screen, waitFor } from '@testing-library/react';
import Notifications from '../Notifications';
import { describe, it, expect, vi } from 'vitest';
import { endpoints } from '../../lib/api';

// Mock API
vi.mock('../../lib/api', () => ({
    endpoints: {
        getEvents: vi.fn()
    }
}));

const mockEvents = [
    {
        id: 'evt_1',
        type: 'subscription.created',
        object_type: 'subscription',
        created_at: '2025-01-01T12:00:00Z'
    },
    {
        id: 'evt_2',
        type: 'invoice.payment_failed',
        object_type: 'invoice',
        created_at: '2025-01-02T12:00:00Z'
    }
];

describe('Notifications Page', () => {
    it('displays loading state initially', async () => {
        endpoints.getEvents.mockReturnValue(new Promise(() => { })); // Hang
        render(<Notifications />);
        expect(screen.getByText('Loading notifications...')).toBeInTheDocument();
    });

    it('renders notifications from API', async () => {
        endpoints.getEvents.mockResolvedValue({ data: { data: mockEvents } });
        render(<Notifications />);

        await waitFor(() => {
            expect(screen.queryByText('Loading notifications...')).not.toBeInTheDocument();
        });

        expect(screen.getByText('New Subscription')).toBeInTheDocument();
        expect(screen.getByText('Payment Failed')).toBeInTheDocument();
    });

    it('displays empty state', async () => {
        endpoints.getEvents.mockResolvedValue({ data: { data: [] } });
        render(<Notifications />);

        await waitFor(() => {
            expect(screen.getByText('No notifications found.')).toBeInTheDocument();
        });
    });
});
