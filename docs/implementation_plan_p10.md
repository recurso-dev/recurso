# Phase 10: Premium UI/UX Implementation Plan

## Goal
Overhaul the existing React frontend to use Tailwind CSS, implementing a premium "Stripe-like" design based on the provided "Stitch" reference structure.

## Components
### 1. Design System (Foundations)
*   **Layout**: `DashboardLayout` with a fixed Sidebar and Top Header.
*   **Navigation**: Sidebar links to Dashboard, Customers, Plans, Subscriptions, Developers, Settings.
*   **Theme**: Slate (50-900) for structure, Indigo-600 for primary actions.

### 2. Dashboard Home (`/`)
*   **Metrics Grid**: Row of cards showing MRR, Active Subscribers, Churn Rate.
*   **Activity Feed**: Simple list of recent events (New Subscription, Payment Failed).

### 3. Entity Pages
*   **Customers**: Data Grid with filters (Status, Date).
*   **Plans**: Card grid or Table view of billing plans.
*   **Subscriptions**: List view with status badges (Active=Green, Canceled=Red).

### 4. Developer Experience
*   **API Keys**: Section in Settings to view/roll API keys.
*   **Webhooks**: List of configured endpoints.

## Execution Order
1.  **Setup**: Install Tailwind (Done). Create `DashboardLayout`.
2.  **Home**: Implement Metrics Cards and basic Chart.
3.  **Customers**: Implement Data Grid.
4.  **Polish**: Transitions, Hover states, Loading skeletons.
