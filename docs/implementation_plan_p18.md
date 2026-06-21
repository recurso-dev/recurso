# Phase 18: UI Polish - Create Screens

Refactor the "Create" screens to match the new "premium" designs provided in `stitch_dashboard_home_screen-update-3`.

## Goal
Improve the aesthetics and UX of the creation flows for Customers, Plans, Subscriptions, and Coupons by implementing the provided high-fidelity HTML/Tailwind designs.

## Proposed Changes

### Frontend

#### [MODIFY] [CreateCustomer.jsx](file:///Users/swapnull/Documents/Workspace/recur-so/frontend/src/pages/CreateCustomer.jsx)
- Adapt layout to match `add_new_customer_screen/code.html`.
- Use the "Slide-Over" panel aesthetic (even if displayed as a page for now).
- Sections: Contact Information, Billing Details.
- Styling: Premium inputs, spacing, and typography.

#### [MODIFY] [CreatePlan.jsx](file:///Users/swapnull/Documents/Workspace/recur-so/frontend/src/pages/CreatePlan.jsx)
- Adapt layout to match `create_new_plan_screen/code.html`.
- Sections: Plan Details (Name, Description), Pricing (Price, Currency, Interval).
- Add "Usage-Based Billing" toggle visual (even if functionality is future).

#### [MODIFY] [CreateSubscription.jsx](file:///Users/swapnull/Documents/Workspace/recur-so/frontend/src/pages/CreateSubscription.jsx)
- Adapt layout to match `create_new_subscription_screen/code.html` (need to check content, assuming it exists similar to others).
- Sections: Customer Selection, Plan Selection, Date.

#### [MODIFY] [CreateCoupon.jsx](file:///Users/swapnull/Documents/Workspace/recur-so/frontend/src/pages/CreateCoupon.jsx)
- Adapt layout to match `create_new_coupon_form/code.html` (need to check content).

#### [MODIFY] [Products.jsx](file:///Users/swapnull/Documents/Workspace/recur-so/frontend/src/pages/Products.jsx)
- Wire up "Create New Product" button to navigate to `/plans/new`.

## Verification Plan

### Manual Verification
- **Create Customer**: Verify the new form looks like the design and creates a customer successfully.
- **Create Plan**: Verify the new form looks like the design and creates a plan successfully.
- **Create Subscription**: Verify the new form looks like the design and creates a subscription successfully.
- **Create Coupon**: Verify the new form looks like the design and creates a coupon successfully.
