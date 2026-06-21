# Implementation Plan - Phase 6: Frontend Admin Dashboard 🖥️

## Goal
Build a premium, "wow-factor" Admin Dashboard for Recurso. This single-page application (SPA) will allow administrators to manage the billing engine visually.

## User Review Required
> [!IMPORTANT]
> **Tech Stack**: We will use **React** (via Vite) for the framework.
> **Styling**: Strictly **Vanilla CSS** with a robust variable system for a "Premium Dark Mode" aesthetic (Glassmorphism, Neon accents).
> **Location**: Source code will reside in a new `frontend/` directory.

## Proposed Changes

### 1. Project Initialization
- Initialize Vite project: `npm create vite@latest frontend -- --template react`
- Setup Proxy in `vite.config.js` to forward `/v1` requests to `http://localhost:8080`.

### 2. Design System (`index.css`)
- Define CSS Variables for:
    - Colors: `surface-dark`, `surface-glass`, `primary-neon`, `text-main`.
    - Spacing & Typography (Inter font).
    - Effects: `glass-blur`, `neon-glow`.

### 3. Core Components
- `Layout`: Sidebar navigation + Main content area.
- `Card`: Glassmorphic containers for data.
- `DataTable`: Reusable table with hover effects for Customers/Plans.
- `Button`: Premium buttons with micro-interactions.

### 4. Features
#### [NEW] `frontend/src/pages/Dashboard.jsx`
- Visualize MRR (Monthly Recurring Revenue) using `recharts`.
- Show "Recent Transactions" list.

#### [NEW] `frontend/src/pages/Customers.jsx`
- List all customers.
- "Add Customer" modal/form.

#### [NEW] `frontend/src/pages/Plans.jsx`
- List active plans.
- "Create Plan" form.

#### [NEW] `frontend/src/auth/AuthProvider.jsx`
- Simple context to store the `API_KEY`.
- "Login Screen" that simply accepts the API Key to store in LocalStorage.

## Verification Plan

### Manual Verification
1.  Run `npm run dev` in `frontend/`.
2.  Log in with the API Secret (`recurso_secret`).
3.  Verify data loads from the Go backend.
4.  Create a plan via UI and verify it appears in the list.
