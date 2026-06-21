# Recurso - "Mega Prompts" for AI Context

Use these prompts to give context to other AI tools (like Stich, v0, Bolt, ChatGPT) about the Recurso product.

## 1. The "Product Architect" Prompt (Full Context)
> **Copy/Paste this to give full context of the system:**

```text
You are an expert Frontend Architect building the UI for "Recurso".

**Product Overview:**
Recurso is a modern, developer-first Billing Engine (Open Source Stripe Alternative). It handles subscriptions, recurring invoices, and usage-based metering for B2B SaaS companies.

**Key Features:**
1.  **Multi-Tenancy:** The platform is SaaS-ready. Users "Register" their own tenant (workspace) and manage their customers completely isolated from others.
2.  **Product Catalog:** Users define "Plans" (e.g., $10/month Gold Plan) with specific currencies and billing intervals.
3.  **Customer CRM:** Manage customers, their billing details, and active subscriptions.
4.  **Subscription Engine:** Logic to handle billing cycles (monthly/yearly), prorations (future), and status transitions (active -> past_due -> canceled).
5.  **Usage Metering:** Ingest API events (e.g., "api_calls") and aggregate them for billing (Pay-as-you-go).
6.  **Coupons:** Discount engine (percent/amount off) handling redemptions and duration (forever/once).
7.  **Financial Ledger:** Integrated with TigerBeetle (Double-Entry) for financial correctness.

**Target Audience:**
Developers and Founders. The UI must be technical, clean, and fast.

**Design System / Vibe:**
*   **Style:** "Stripe-like". Clean, minimalistic, professional.
*   **Color Palette:** Slate/Gray scale for structure. Indigo/Violet for primary actions. Emerald/Red for status indicators (Paid/Failed).
*   **Typography:** Inter or San Francisco. High readability.
*   **Components:** Data Tables (dense), Status Badges (Pills), Slide-over panels for details, Line Charts for Analytics (MRR).

**Current Tech Stack:**
*   **Backend:** Go (Gin), PostgreSQL, TigerBeetle.
*   **Frontend:** React (Vite), Tailwind CSS (Required), Lucide React (Icons), Recharts (Analytics).

**Your Task:**
Build a premium, responsive Dashboard UI. Focus on:
1.  **Sidebar Navigation**: (Home, Customers, Plans, Subscriptions, Invoices, Developers, Settings).
2.  **Dashboard Home**: Metrics cards (MRR, Active Subs) + Recent Activity Table.
3.  **Data Grids**: High-density lists with filters and pagination.
```

## 2. The "Design System" Prompt (Visuals Only)
> **Use this if you just want UI generation:**

```text
Create a modern B2B SaaS Dashboard for a Billing Platform called "Recurso".
Use Tailwind CSS.
Visual constraints:
1.  **Background**: Light gray (#f9fafb) for the app background, White (#ffffff) for cards/panels.
2.  **Primary Color**: Indigo-600.
3.  **Layout**: Fixed sidebar on the left (dark or light), main content area on the right.
4.  **Shadows**: Subtle, diffuse shadows (shadow-sm, shadow-md) to create depth.
5.  **Borders**: Thin, crisp borders (border-gray-200).
6.  **Typography**: Sans-serif, deeply hierarchical (Text-gray-900 for headings, Text-gray-500 for secondary).
7.  **Interactive**: Hover states on all rows and buttons.

Generate a "Customer List" page with:
-   Header with "Add Customer" button (Primary).
-   Search bar and filters row.
-   A table showing: Name, Email, Status (Active/Churned), MRR, Joined Date.
-   Use Lucide icons for actions.
```
