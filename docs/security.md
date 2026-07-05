# Security Posture

How Recurso handles sensitive data, what's in and out of scope, and how the
deployment is hardened. This page describes the actual implementation —
each claim maps to code in this repository.

## Card data and PCI scope

**Recurso never touches primary account numbers (PANs).** All card
collection and charging happens inside Stripe and Razorpay's certified
environments; Recurso stores only:

- Gateway tokens/identifiers (Stripe payment method and subscription IDs,
  Razorpay token and customer IDs)
- Display metadata: card brand, last 4 digits, expiry month/year

This keeps a self-hosted Recurso deployment out of PCI DSS scope for card
storage (SAQ-A-style posture). Do not modify the gateways to proxy raw card
input through the API — that would change your compliance obligations.

## Credentials

- **API keys are bcrypt-hashed at rest** (`internal/adapter/db/
  tenant_repository.go`). The plaintext key is returned exactly once at
  creation; the database stores the hash plus an 8-character prefix for
  lookup. List endpoints return masked keys.
- **Customer portal magic links** are cryptographically random tokens,
  single-use, with a hard expiry; portal sessions are separate random
  tokens with their own expiry. Tokens are only ever delivered by email
  (the response-body debug link is development-only, gated on
  `APP_ENV=development`).
- **Tenant passwords** (dashboard registration) are bcrypt-hashed.
- Payment gateway and SMTP credentials come exclusively from environment
  variables / your secrets manager — nothing is hardcoded.

## Tenant isolation

Recurso is multi-tenant at the row level: every tenant-owned table carries
`tenant_id`, repository queries filter on it, and the tenant is derived
from the authenticated API key via a typed context key (not a raw string).
The small number of deliberately cross-tenant lookups (Stripe webhook
subscription resolution, portal login by email) are documented at the
method level with the reason, and guarded by unique indexes or downstream
scoping.

## Inbound webhooks

Stripe events are verified with `webhook.ConstructEvent` (signature check
against `STRIPE_WEBHOOK_SECRET`); Razorpay events are verified with
HMAC-SHA256 against `RAZORPAY_WEBHOOK_SECRET`. Unverified payloads are
rejected before any processing.

## Transport and runtime hardening

- TLS terminates at your reverse proxy or the K8s ingress (cert-manager
  wiring included); the API listens on plain HTTP only inside the network.
- Containers run as non-root (API: dedicated user on Alpine; dashboard:
  unprivileged nginx, uid 101) with read-only root filesystems in K8s,
  all capabilities dropped, seccomp `RuntimeDefault`, no service account
  token, and a default-deny ingress NetworkPolicy.
- Rate limiting: global per-key limits plus stricter per-IP limits on
  public endpoints (registration, portal auth, webhooks).

## Supply chain

Every commit runs `govulncheck` (fails on any reachable vulnerability) and
Trivy filesystem/secret/misconfig scans; container images are Trivy-scanned
before publish. Releases are tagged, version-stamped builds published to
GHCR.

## Financial integrity

Money movements are double-entry ledger transactions (PostgreSQL, with
optional TigerBeetle mirroring). Ledger write failures surface to callers
instead of being swallowed; negative amounts are rejected before posting.
Invoice arithmetic, tax splits, and payment application are covered by the
unit test suite.

## Reporting a vulnerability

Please do not open public issues for suspected vulnerabilities. Email
**security@recurso.dev** with reproduction details; we aim to acknowledge
within 48 hours. (Self-hosters: security fixes are announced in release
notes — watch the repository.)
