# Phase 11: UI Completion & Dependency Updates

## Goal
Fix broken/missing pages (Subscriptions, Invoices, Developers, Settings) by implementing them according to the reference design. Update project dependencies to the latest stable versions while ensuring build stability.

## Proposed Changes

### Dependencies
- [ ] Update `vite`, `postcss`, `autoprefixer` to latest stable versions.
- [ ] Ensure `tailwindcss` remains at v3.4.x (latest stable v3) to prevent v4 alpha breakage.
- [ ] Update `react` / `react-dom` if newer minor versions exist.

### Frontend
#### [NEW] `src/pages/Subscriptions.jsx`
- Implement based on `stitch_dashboard_home_screen/subscriptions_data_grid/code.html`.
- Features: Data grid with Status pills, plan details, and filtering.

#### [NEW] `src/pages/Invoices.jsx`
- Implement based on design patterns from Customers/Subscriptions (as specific `invoices_data_grid` is missing).
- Features: List of invoices with status (Paid, Due, Void), amounts, and download actions.

#### [NEW] `src/pages/Developers.jsx`
- Implement based on `stitch_dashboard_home_screen/developers_settings/code.html` (or infer from Settings).
- Features: API Key management, Webhook endpoints.

#### [NEW] `src/pages/Settings.jsx`
- Implement based on `stitch_dashboard_home_screen/developers_settings/code.html`.
- Features: General settings, Team management.

#### [MODIFY] `src/App.jsx`
- Update routes to point to the new components instead of placeholders.

## Verification Plan
### Automated Tests
- Run `npm run build` to ensure no build errors with new dependencies.
### Manual Verification
- Navigate to all new pages.
- Verify "Stitch" design fidelity (fonts, colors, layout).
- Verify responsiveness.
