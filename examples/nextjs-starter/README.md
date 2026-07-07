# Recurso Next.js Starter

A minimal but real Next.js (App Router) SaaS starter wired to the
[Recurso](https://recurso.dev) billing engine. It shows the end-to-end flow:
list plans, sign a customer up, show their subscription usage with entitlement
headroom, and gate a page behind an entitlement check.

Every Recurso call runs **server-side** (server components + a route handler),
so your secret API key is never shipped to the browser.

## What it demonstrates

| Page          | Route       | Recurso API                              |
| ------------- | ----------- | ---------------------------------------- |
| Pricing       | `/pricing`  | `GET /v1/plans`                          |
| Fake signup   | `/pricing`  | `POST /v1/customers`, `POST /v1/subscriptions` |
| Account       | `/account`  | `GET /v1/subscriptions/{id}/usage`       |
| Feature gate  | `/feature`  | `GET /v1/entitlements/check`             |

After signup, the new customer/subscription ids are stored in `httpOnly`
cookies so the account and feature pages can identify the session. A real app
would attach those ids to its own authenticated user record.

## Prerequisites

Run the Recurso stack from the main repo and seed the demo tenant:

```bash
# in the recur-so repo root
make demo
```

That starts the API on `http://localhost:8080` and provisions the test API key
`sk_test_12345` along with sample plans.

## Run the starter

```bash
cd examples/nextjs-starter
cp .env.example .env.local     # defaults already point at make demo + sk_test_12345
npm install
npm run dev                    # http://localhost:3000
```

Open http://localhost:3000, go to **Pricing**, complete the signup, then visit
**Account** to see usage.

## Environment

| Variable           | Purpose                                                    |
| ------------------ | --------------------------------------------------------- |
| `RECURSO_API_URL`  | Base URL of the Recurso API (default `http://localhost:8080`) |
| `RECURSO_API_KEY`  | **Secret** API key. Server-side only — never expose it.   |

> The key is read only in `lib/recurso.ts`, which is imported exclusively by
> server components and the route handler. Do not import it from a
> `"use client"` component and never prefix it with `NEXT_PUBLIC_`.

## Notes

- **Usage table empty?** A freshly created subscription has no usage yet.
  Report an event with `POST /v1/usage/events` and reload `/account`.
- **Feature page shows "Locked"?** Expected out of the box — the demo seed
  doesn't grant the `advanced_reports` feature. Grant it to a plan with
  `PUT /v1/plans/{id}/entitlements`, then reload.
- **No SDK dependency.** This starter uses plain `fetch`. A typed Node SDK
  lives at `sdk/node` in the main repo; it isn't published to npm yet, so drop
  it in once it ships.

## Build

```bash
npm install
npm run build
```
