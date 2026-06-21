# Implementation Plan - Phase 9: Developer Experience (SDKs) 📦

## Goal
Make **Recurso** easy to integrate for developers by providing a typed, idiomatic Node.js SDK (and an OpenAPI spec for other languages). This dramatically lowers the barrier to entry for your "customers".

## Proposed Changes

### 1. API Specification
#### [NEW] `api/openapi.yaml`
- Author a complete OpenAPI 3.0 specification describing our endpoints:
    - `POST /auth/register`
    - `POST /v1/customers`
    - `POST /v1/plans`
    - `POST /v1/subscriptions` (including `coupon_code`)
    - `POST /v1/usage/events`
    - `GET /v1/analytics/mrr`

### 2. SDK Generation (Node.js/TypeScript)
#### [NEW] `sdk/node/`
- We will use `openapi-typescript-codegen` (lightweight, popular) or write a clean wrapper manually if generation is too clunky.
- **Decision**: Manually authoring a lightweight wrapper is often better for a *starting* startup to ensure idiomatic feel ("Stripe-like"), but generation is faster.
- **Hybrid Approach**: We will generate the *Types* (`sdk/node/src/types.ts`) from OpenAPI, but write the *Client* (`recurso.ts`) manually to ensure a premium developer experience (`recurso.customers.create(...)`).

### 3. Example Code
#### [NEW] `examples/node-quickstart/index.js`
- A script showing a developer how to sign up, get a key, and charge a user.

## Verification Plan
1.  **Generate**: Run the generation script.
2.  **Run Example**: Execute `node examples/node-quickstart/index.js`.
3.  **Success**: The script should print "Subscription Created: sub_123".
