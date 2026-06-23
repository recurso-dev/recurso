---
sidebar_position: 1
---

# Introduction

Welcome to **Recurso** -- the open-source billing engine for SaaS companies.

Recurso gives you complete control over subscription billing, invoicing, usage metering, and revenue tracking. It is built with Go and PostgreSQL for performance, and uses TigerBeetle for an immutable financial ledger. Self-host it on your own infrastructure, or use the managed cloud offering.

## Who is Recurso for?

- **SaaS founders** who want billing that scales without a revenue tax.
- **Engineering teams** who need a programmable billing engine they can extend and integrate deeply.
- **Finance teams** who require audit-ready ledgers and India-compliant tax handling (GST, TDS, e-invoicing) out of the box.

## What can you do with Recurso?

- **Subscriptions** -- Recurring billing with trials, upgrades, downgrades, and proration.
- **Invoicing** -- Automatic invoice generation, PDF delivery, and tax calculation.
- **Usage Metering** -- Ingest events in real time for metered and hybrid billing models.
- **Entitlements** -- Feature gating and access control tied to plans.
- **Dunning** -- AI-powered smart retries to recover failed payments automatically.
- **Analytics** -- Real-time MRR, churn, and revenue dashboards.

## Getting Started

The fastest way to get up and running is the [Quickstart Guide](/getting-started/quickstart). You will have a working billing API in under five minutes.

## Key Concepts

### Tenants
Recurso is multi-tenant by design. Every resource belongs to a tenant (your organization), and you authenticate using tenant-scoped API keys.

### Plans & Prices
Define what you sell. A Plan has a billing interval (monthly, yearly, or one-time) and one or more price tiers.

### Customers
The entities purchasing your plans. Customers hold billing details, tax IDs (including GSTIN), and subscription state.

### Subscriptions
The link between a Customer and a Plan. Subscriptions drive the billing cycle, generate invoices, and manage lifecycle transitions.

## SDKs

Official client libraries:

- [Node.js SDK](https://github.com/recurso/node-sdk)
- [Go SDK](https://github.com/recurso/go-sdk)
- [Python SDK](https://github.com/recurso/python-sdk)

## Need Help?

- Browse the [API Reference](/api-reference/plans) for endpoint details.
- Join the conversation on [GitHub Discussions](https://github.com/recur-so/recurso/discussions).
- Report issues on [GitHub](https://github.com/recur-so/recurso/issues).
