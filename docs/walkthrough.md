# Walkthrough - Phase 12: Frontend API Integration

I have successfully connected the React frontend to the Go backend! The application now fetches real data instead of using static placeholders.

## 🚀 Key Changes

### 1. Backend: List Endpoints Added
To support the UI data grids, I implemented `GET` (List) endpoints for all core resources.
*   `GET /v1/plans` - List all plans for the tenant.
*   `GET /v1/customers` - List all customers.
*   `GET /v1/subscriptions` - List active and inactive subscriptions.
*   `GET /v1/invoices` - List invoices with status.

### 2. Frontend: Axios API Client
Created `src/lib/api.js` to handle:
*   Base URL configuration (`/v1`).
*   **Authentication**: Automatically attaches the `Authorization: Bearer <token>` header from `localStorage`.

### 3. Page Integration
Refactored the following pages to fetch data on load:
*   **Dashboard**: Shows real MRR and Active Subscription count.
*   **Plans**: Lists plans from the backend.
*   **Customers**: Lists customers.
*   **Subscriptions**: Lists subscriptions.
*   **Invoices**: Lists invoices.

### 4. Development Login
Updated the **Login** page:
*   Entering `recurso_secret` now correctly logs you in as the "Dev Tenant" (ID: 1).
*   Stores the key in `localStorage` for the API client to use.

## 🧪 How to Verify

1.  **Ensure Backend is Running**:
    ```bash
    go run cmd/api/main.go
    ```
2.  **Login**:
    *   Go to `/login`.
    *   Enter `recurso_secret`.
    *   Click "Access Dashboard".
3.  **Browse**:
    *   Navigate to **Plans**, **Customers**, etc.
    *   You should see "No plans found" (or data if you seeded the DB).
    *   Create a plan via `POST /v1/plans` (using Postman/Curl) or wait for the "New Plan" UI implementation (P13).

## 📸 Screenshots

*(Use the app to see the live data loading!)*
