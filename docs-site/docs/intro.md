---
sidebar_position: 1
---

# Introduction

Welcome to the **Recurso Developer Documentation**.

Recurso is the open-source billing engine designed for high-growth SaaS. It handles:
- **Subscriptions**: Recurring billing, proration, and lifecycle management.
- **Invoicing**: PDF generation, email delivery, and dunning.
- **Metering**: Usage-based billing with real-time event ingestion.
- **Entitlements**: Feature gating and access control.

## Getting Started

To get started with Recurso, we recommend following our [Quickstart Guide](/getting-started/quickstart).

## Key Concepts

### Tenants
Recurso is multi-tenant by design. Every resource belongs to a specific tenant (your organization). You interact with the API using specific API Keys for your tenant.

### Plans & Products
Define what you sell using Plans. A Plan consists of a billing interval (monthly/yearly) and a price.

### Customers
The entities purchasing your plans. Customers hold billing information, tax IDs, and subscription states.

### Subscriptions
The link between a Customer and a Plan. Subscriptions manage the billing cycle and invoices.

## SDKs

We provide official SDKs for:
- [Node.js](https://github.com/recurso/node-sdk)
- [Go](https://github.com/recurso/go-sdk)
- [Python](https://github.com/recurso/python-sdk)
