---
sidebar_position: 9
---

# Revenue Recognition

Recurso automatically handles revenue recognition in compliance with ASC 606 principles. When an invoice is paid, the system creates a recognition schedule that allocates revenue across the subscription period on a monthly basis. A background worker processes due recognition events daily, moving revenue from deferred to recognized.

## Key Concepts

### ASC 606 Basics

Under ASC 606, revenue is recognized when the performance obligation is satisfied -- not when cash is collected. For subscription businesses, this means:

- **Deferred revenue**: Cash collected upfront for services not yet delivered. For example, if a customer pays $1,200 for a 12-month subscription on January 1, only $100 is recognized in January. The remaining $1,100 is deferred.
- **Recognized revenue**: Revenue earned as the service is delivered over time. Each month, $100 moves from deferred to recognized.

This distinction is critical for accurate financial reporting, investor communications, and audit readiness.

### How Recurso Handles It

1. **Invoice paid**: When an invoice transitions to `paid` status, Recurso creates a `RevenueSchedule` that spans the subscription period (from `current_period_start` to `current_period_end`).
2. **Schedule splitting**: The total invoice amount is divided into monthly `RecognitionEvent` records. Each event has a `recognition_date` set to the first day of its respective month.
3. **Daily worker**: The `RevRecWorker` runs every 24 hours. It queries for `pending` recognition events whose `recognition_date` is on or before today and marks them as `recognized`. If a ledger is configured (TigerBeetle or PG-based), the worker also records the corresponding ledger transaction.
4. **Cancellations**: If a subscription is canceled mid-period, the remaining unrecognized events on the schedule are not processed. The schedule status is updated to `canceled`.

### The Revenue Schedule Object

```json
{
  "id": "e1f2a3b4-c5d6-7890-ef12-abcdef123456",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "invoice_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
  "subscription_id": "d4e5f6a7-b8c9-0123-def0-1234567890ab",
  "total_amount": 1200000,
  "currency": "USD",
  "start_date": "2026-01-01T00:00:00Z",
  "end_date": "2026-12-31T23:59:59Z",
  "status": "active",
  "created_at": "2026-01-01T12:00:00Z",
  "updated_at": "2026-01-01T12:00:00Z"
}
```

| Field              | Type    | Description                                                  |
|--------------------|---------|--------------------------------------------------------------|
| `id`               | string  | Unique schedule identifier (UUID).                           |
| `tenant_id`        | string  | The tenant this schedule belongs to.                         |
| `invoice_id`       | string  | The paid invoice that triggered this schedule.               |
| `subscription_id`  | string  | The associated subscription (optional for one-off invoices). |
| `total_amount`     | integer | Total amount in cents to be recognized over the period.      |
| `currency`         | string  | 3-letter ISO currency code.                                  |
| `start_date`       | string  | Start of the recognition period (ISO 8601).                  |
| `end_date`         | string  | End of the recognition period (ISO 8601).                    |
| `status`           | string  | `active` or `canceled`.                                      |

### The Recognition Event Object

```json
{
  "id": "f2a3b4c5-d6e7-8901-f012-3456789abcde",
  "revenue_schedule_id": "e1f2a3b4-c5d6-7890-ef12-abcdef123456",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "amount": 100000,
  "recognition_date": "2026-06-01T00:00:00Z",
  "status": "recognized",
  "ledger_tx_id": "a0b1c2d3-e4f5-6789-0abc-def012345678",
  "created_at": "2026-01-01T12:00:00Z"
}
```

| Field                 | Type    | Description                                              |
|-----------------------|---------|----------------------------------------------------------|
| `id`                  | string  | Unique event identifier (UUID).                          |
| `revenue_schedule_id` | string  | The parent schedule.                                     |
| `amount`              | integer | Amount in cents to recognize for this period.            |
| `recognition_date`    | string  | The date this revenue should be recognized.              |
| `status`              | string  | `pending`, `recognized`, or `failed`.                    |
| `ledger_tx_id`        | string  | Reference to the ledger transaction (if ledger is enabled). |

## Revenue Recognition Report

The report endpoint returns a summary of recognized and deferred revenue for a given month.

### Get Report

**GET** `/v1/finance/revrec/report`

```bash
curl "https://api.recurso.dev/v1/finance/revrec/report?month=6&year=2026" \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "data": {
    "month": 6,
    "year": 2026,
    "recognized_amount": 4850000,
    "deferred_amount": 7230000,
    "schedules_count": 142,
    "events_processed": 142,
    "events_pending": 0,
    "currency": "USD"
  }
}
```

| Query Parameter | Type    | Required | Description                       |
|-----------------|---------|----------|-----------------------------------|
| `month`         | integer | Yes      | Month number (1-12).              |
| `year`          | integer | Yes      | Four-digit year (e.g., 2026).     |

### Understanding the Report

- **`recognized_amount`**: Total revenue (in cents) that has been earned and recognized for the specified month. This is the sum of all `RecognitionEvent` records with status `recognized` whose `recognition_date` falls within the month.
- **`deferred_amount`**: Total revenue (in cents) that has been collected but not yet recognized. This represents future performance obligations.
- **`schedules_count`**: Number of active revenue schedules that have events in this month.
- **`events_processed`**: Number of recognition events that have been successfully processed.
- **`events_pending`**: Number of recognition events still awaiting processing.

## Example: Annual Subscription

Consider a customer who pays $12,000/year ($1,000/month) on June 1, 2026.

1. **Invoice paid**: Invoice `inv_001` for $12,000 is marked as paid.
2. **Schedule created**: A `RevenueSchedule` is created with `total_amount: 1200000`, `start_date: 2026-06-01`, `end_date: 2027-05-31`.
3. **Events generated**: 12 `RecognitionEvent` records are created, one per month, each for $1,000 (100,000 cents):
   - June 2026: 100,000 cents -- recognized immediately by the daily worker
   - July 2026: 100,000 cents -- pending until July 1
   - August 2026: 100,000 cents -- pending until August 1
   - ... and so on through May 2027
4. **June report**: Shows `recognized_amount: 100000` from this subscription (plus any other schedules) and `deferred_amount: 1100000` (the remaining 11 months).

## Background Worker

The `RevRecWorker` runs on a 24-hour cycle. On each tick it:

1. Queries for all `RecognitionEvent` records where `status = 'pending'` and `recognition_date <= now()`.
2. For each event, marks it as `recognized`.
3. If a ledger (TigerBeetle or PostgreSQL-based) is configured, records a debit to the Deferred Revenue account and a credit to the Recognized Revenue account.
4. If processing fails for any event, that event is marked as `failed` and will be retried on the next worker cycle.

No manual intervention is required. The worker handles all state transitions automatically.
