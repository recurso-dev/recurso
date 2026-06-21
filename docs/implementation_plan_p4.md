# Implementation Plan - Phase 4: Security & Infrastructure 🔒

## Goal
Harden the application for production by adding Authentication (JWT) and Containerization (Docker).

## User Review Required
> [!IMPORTANT]
> **Authentication**: We will use a simple "API Key" or "Admin Token" approach for machine-to-machine communication (S2S) and a JWT flow if we were doing user sessions. Given this is a B2B Billing Engine, **API Keys** are often more appropriate for the `/v1/` routes used by the Tenant's backend.
> **Decision**: We will implement **JWT** for the *Dashboard/Portal* and **API Keys** for the *Ingestion API*. (For simplicity in this phase, we'll start with a shared "Admin Secret" for API protection).

## Proposed Changes

### 1. Authentication Middleware
Protect critical endpoints.

#### [NEW] `internal/adapter/middleware/auth.go`
- Middleware to check `Authorization: Bearer <token>` or `X-API-Key`.
- For P4, we will use a static `API_SECRET` env var to sign/verify tokens or act as the key.

#### [MODIFY] `cmd/api/main.go`
- Apply middleware to `/v1/` routes.
- Exempt `/health`, `/portal` (public for demo, or maybe secure it?), `/webhooks` (has its own signature verification).

### 2. Dockerization
Make it runnable anywhere.

#### [NEW] `Dockerfile`
- Multi-stage build (Build -> Distroless/Alpine).

#### [NEW] `docker-compose.yml`
- Services: `app`, `postgres`.
- Volume for Postgres data.

## Verification Plan

### Automated Tests
1.  **Auth**: `curl -v /v1/plans` (No Header) -> `401 Unauthorized`.
2.  **Auth**: `curl -v /v1/plans -H "Authorization: Bearer ..."` -> `201 Created`.

### Manual Verification
1.  **Docker**:
    ```bash
    docker-compose up --build
    ```
    Access `localhost:8080/health`.
