# Recurso: Research Summary & Architectural Overview

## 1. Executive Summary
Recurso is an **AI-Native Subscription Revenue Platform** designed for the **Global Market**, with specialized deep integration for the **Indian ecosystem**. It aims to solve the "Compliance as Debt" and "Success Tax" problems of incumbent platforms (Chargebee/Stripe) by offering a comprehensive, globally-capable yet locally-compliant, architecturally correct, and AI-driven solution.

**Key Differentiators:**
- **Global & Local Compliance:** Native support for RBI e-mandates and data localization (India), while maintaining global billing standards.
- **Correctness First:** "Write-Last, Read-First" consistency using a double-entry ledger (TigerBeetle).
- **AI-Native:** Reinforcement Learning for smart retries and GenAI (Text-to-SQL) for analytics.
- **Fair Pricing:** Transaction/feature-based rather than revenue-percentage based.

## 2. Technical Stack

| Component | Technology | Role |
| :--- | :--- | :--- |
| **Service Layer** | **Go (Golang)** | API Gateway, Orchestrator, Webhooks (High concurrency) |
| **Core Ledger** | **Rust / TigerBeetle** | Immutable Double-Entry Ledger (Performance & Safety) |
| **Metadata DB** | **PostgreSQL** | System of Reference (Users, Plans, Configs) |
| **Financial DB** | **TigerBeetle** | System of Record (Balances, Transfers) |
| **Analytics DB** | **ClickHouse** | Timeseries metrics, Metered billing events |
| **Event Bus** | **Kafka / Redpanda** | Async communication, Decoupling services |
| **Orchestrator** | **Temporal** | Durable workflows for Subscription Lifecycles (Sleep for days) |
| **Infrastructure** | **Kubernetes (EKS)** | Multi-AZ in `ap-south-1` (Mumbai) |

## 3. Core Architectural Patterns

### 3.1 "Write-Last, Read-First" Consistency
To ensure the ledger (TigerBeetle) and metadata (Postgres) never drift:
1.  **Read First (Postgres):** Validate state (e.g., subscription is active).
2.  **Generate ID:** Create deterministic Transaction ID.
3.  **Write Last (TigerBeetle):** Commit financial movement. This is the point of no return.
4.  **Update Reference (Postgres):** If ledger write succeeds, update invoice status.

### 3.2 Subscription Mathematics (Exact-Time Proration)
- **Precision:** Seconds-level accuracy for upgrades/downgrades.
- **Formula:** Calculates unused credit from old plan vs. remaining charge for new plan.
- **Currency:** Integer-based (Paisa) to avoid floating-point errors.

### 3.3 The India Payment Stack (UPI AutoPay)
- **Asynchronous Model:** Handles the complex state machine of UPI mandates (Initiated -> Pending -> Active).
- **T-24 Notification:** Automated scheduler to send mandatory pre-debit notifications 24 hours before charge.
- **Tokenization:** Stores only token references (CoF) to remain outside PCI-DSS scope.

## 4. AI & Intelligence
- **Smart Retries:** Contextual Bandit algorithm (RL) to determine optimal retry times based on bank health, time of day, etc.
- **GenAI Analytics:** RAG pipeline allowing "Text-to-SQL" queries on financial data (e.g., "Show churn rate for last month").

## 5. Implementation Roadmap
- **Phase 1 (Wk 1-8):** Core Ledger (TigerBeetle setup), Subscription Engine (Time modeling), Basic APIs.
- **Phase 2 (M 3-5):** Payment Integration (Razorpay, UPI AutoPay), Tokenization.
- **Phase 3 (M 5-7):** Compliance (GST E-invoicing), Proration logic.
- **Phase 4 (M 7-10):** AI Layer (Smart Retries, GenAI).
