# Recurso: Feature Specifications (Priority 2 & 3)

This document details the advanced features that differentiate Recurso from a basic billing tool, transforming it into an intelligent Revenue Operating System.

## 1. AI & Intelligence Layer (Priority 2)

### 1.1 Smart Retries (Reinforcement Learning)
**Problem**: Static retry schedules (Day 1, 3, 7) are inefficient.
**Solution**: Use a Contextual Bandit model to predict the probability of success.
- **Inputs (Context)**:
    - `BIN` (Bank Identification Number) -> Identifies Issuing Bank.
    - `Error Code` -> "Insufficient Funds" vs "Generic Decline".
    - `Time` -> Hour of day, Day of week.
    - `History` -> Success rate of this card previously.
- **Action Space**: Retry Now, Retry in 6h, Retry Tomorrow 10 AM, Glue to Payday (1st/30th).
- **Feedback**: Success (+1) vs Failure (-Cost).

### 1.2 "Ask Data" Analytics (GenAI)
**Problem**: Founders can't write SQL. Dashboards are static.
**Solution**: Text-to-SQL Interface.
- ** Architecture**:
    - **Step 1**: User asks "Who are my top 10 churned customers from Mumbai?"
    - **Step 2**: LLM (GPT-4o) receives sanitized schema (Table names, Columns) + Question.
    - **Step 3**: LLM generates SQL: `SELECT email FROM customers WHERE status='canceled' AND city='Mumbai' LIMIT 10`.
    - **Step 4**: Safety Guard (Regex) checks for `DROP`, `DELETE` or PII modifications.
    - **Step 5**: Execute on Read-Replica DB -> Return JSON -> Render Table/Chart.

---

## 2. Customer Self-Service (Priority 2)

### 2.1 Customer Portal
A secure, white-labeled web portal for subscribers.
- **Authentication**: Passwordless (Magic Link sent to email).
- **capabilities**:
    - **Download Invoices**: PDF history.
    - **Update Payment Method**: Add new card, set default.
    - **Subscription Management**: Change Plan (Upgrade/Downgrade), Pause, Cancel (with reason survey).
    - **Account Details**: Update tax ID, billing address.

---

## 3. Marketing Tools (Priority 2)

### 3.1 Coupons & Discounts
- **Rules Engine**:
    - `Code`: STRING (e.g., "BLACKFRIDAY").
    - `Type`: Percentage (`20%`) or Fixed (`$10`).
    - `Duration`: `Once`, `Forever`, or `Repeating` (e.g., for 3 months).
    - `Redemption Limit`: Max 100 uses globally, or 1 per customer.
    - `Applicability`: Specific Plans or Global.

---

## 4. Advanced/Enterprise Features (Priority 3)

### 4.1 Revenue Recognition (RevRec)
**Problem**: Cash in bank != Revenue earned (Accrual Accounting).
**Solution**: ASC-606 / IFRS 15 Engine.
- **Logic**:
    - User pays $1200 for Yearly Plan on Jan 1.
    - **Jan 1 Ledger**: Debit Cash $1200, Credit Deferred Revenue $1200.
    - **Jan 31 (Run)**: Debit Deferred Revenue $100, Credit Recognized Revenue $100.
- **Output**: Monthly "Revenue Waterfall" report.

### 4.2 Multi-Entity / Multi-Region
- **Structure**: One Login -> Multiple Organization IDs.
- **Data Isolation**: Logical separation (Row-level security).
- **Consolidated Reporting**: "Group View" dashboard aggregating revenue across US and India entities.

### 4.3 Quotes
- **Workflow**:
    1.  Sales rep creates `Quote` (Draft terms, custom pricing).
    2.  Emails PDF to Prospect.
    3.  Prospect clicks "Accept" on web view.
    4.  System converts `Quote` -> `Subscription` + `Invoice`.

### 4.4 ERP Integrations
- **Sync**: Daily batch jobs to push:
    - Invoices & Credit Notes -> QuickBooks/Xero/NetSuite.
    - Payments -> Bank Reconciliation Module.
