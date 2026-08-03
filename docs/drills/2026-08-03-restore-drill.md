# Restore drill — 2026-08-03

**Result: PASS**

Proves the backup restores completely and the restored books still balance.
Produced by [`scripts/restore_drill.sh`](../../scripts/restore_drill.sh); every
number below is regenerable by re-running it.

| Metric | Value |
|---|---|
| Dump size | 972K |
| Dump time | 1s |
| **Restore time (RTO core)** | **0s** |
| Tenants restored | 97 |
| Ledger transactions restored | 3119 |
| Invoice money checksum (order-independent md5) | OK |
| Double-entry conservation, global (Σdebits = Σcredits) | OK (237458534 = 237458534) |
| Tenants with unbalanced books after restore | 0 |

## Per-table row-count parity

| Table | Source | Restored | Status |
|---|---|---|---|
| `accounting_connections` | 0 | 0 | OK |
| `accounting_entity_mappings` | 0 | 0 | OK |
| `accounting_sync_log` | 0 | 0 | OK |
| `api_keys` | 0 | 0 | OK |
| `audit_logs` | 0 | 0 | OK |
| `billable_metrics` | 0 | 0 | OK |
| `cancel_flow_sessions` | 0 | 0 | OK |
| `cancel_flow_steps` | 0 | 0 | OK |
| `cancel_flows` | 0 | 0 | OK |
| `card_expiry_notifications` | 0 | 0 | OK |
| `churn_alerts` | 0 | 0 | OK |
| `churn_feature_snapshots` | 0 | 0 | OK |
| `consents` | 0 | 0 | OK |
| `coupons` | 96 | 96 | OK |
| `credit_note_applications` | 0 | 0 | OK |
| `credit_notes` | 179 | 179 | OK |
| `customers` | 685 | 685 | OK |
| `dunning_campaign_executions` | 0 | 0 | OK |
| `dunning_campaign_steps` | 0 | 0 | OK |
| `dunning_campaigns` | 0 | 0 | OK |
| `dunning_history` | 0 | 0 | OK |
| `dunning_weights` | 0 | 0 | OK |
| `email_verification_tokens` | 0 | 0 | OK |
| `entities` | 97 | 97 | OK |
| `entity_invoice_sequences` | 97 | 97 | OK |
| `eu_einvoices` | 0 | 0 | OK |
| `event_deliveries` | 0 | 0 | OK |
| `events` | 0 | 0 | OK |
| `gateway_connections` | 0 | 0 | OK |
| `gifts` | 210 | 210 | OK |
| `import_external_refs` | 0 | 0 | OK |
| `inbound_webhook_events` | 0 | 0 | OK |
| `integration_connections` | 0 | 0 | OK |
| `invoice_disputes` | 0 | 0 | OK |
| `invoice_items` | 967 | 967 | OK |
| `invoices` | 1707 | 1707 | OK |
| `ledger_accounts` | 1157 | 1157 | OK |
| `ledger_transactions` | 3119 | 3119 | OK |
| `magic_links` | 0 | 0 | OK |
| `mandates` | 0 | 0 | OK |
| `mcp_settings` | 0 | 0 | OK |
| `mfa_backup_codes` | 0 | 0 | OK |
| `mfa_login_tokens` | 0 | 0 | OK |
| `mrr_snapshots` | 0 | 0 | OK |
| `nexus_alerts` | 0 | 0 | OK |
| `offline_payments` | 0 | 0 | OK |
| `organizations` | 0 | 0 | OK |
| `password_reset_tokens` | 0 | 0 | OK |
| `payment_attempts` | 0 | 0 | OK |
| `plan_charges` | 0 | 0 | OK |
| `plan_entitlements` | 0 | 0 | OK |
| `plans` | 192 | 192 | OK |
| `portal_sessions` | 0 | 0 | OK |
| `precharge_notifications` | 0 | 0 | OK |
| `prices` | 192 | 192 | OK |
| `progressive_billing_watermarks` | 0 | 0 | OK |
| `quotes` | 226 | 226 | OK |
| `recognition_events` | 793 | 793 | OK |
| `recovered_payments` | 0 | 0 | OK |
| `referrals` | 0 | 0 | OK |
| `revenue_schedules` | 740 | 740 | OK |
| `schema_migrations` | 1 | 1 | OK |
| `sessions` | 0 | 0 | OK |
| `sso_connections` | 0 | 0 | OK |
| `sso_consumed_assertions` | 0 | 0 | OK |
| `subscription_addons` | 0 | 0 | OK |
| `subscriptions` | 685 | 685 | OK |
| `tax_registrations` | 0 | 0 | OK |
| `telemetry_instance` | 0 | 0 | OK |
| `tenant_eu_config` | 0 | 0 | OK |
| `tenant_gst_configs` | 0 | 0 | OK |
| `tenant_irp_configs` | 0 | 0 | OK |
| `tenant_tax_nexus` | 0 | 0 | OK |
| `tenant_us_tax_config` | 0 | 0 | OK |
| `tenants` | 97 | 97 | OK |
| `unbilled_charges` | 0 | 0 | OK |
| `us_nexus_thresholds` | 47 | 47 | OK |
| `usage_alerts` | 0 | 0 | OK |
| `usage_events` | 0 | 0 | OK |
| `usage_ratings` | 0 | 0 | OK |
| `user_oauth_identities` | 0 | 0 | OK |
| `users` | 0 | 0 | OK |
| `virtual_accounts` | 0 | 0 | OK |
| `waitlist_signups` | 0 | 0 | OK |
| `wallet_transactions` | 0 | 0 | OK |
| `wallets` | 0 | 0 | OK |
| `webhook_endpoints` | 0 | 0 | OK |

## Method

1. `pg_dump -Fc` of the source database.
2. `pg_restore` into a **fresh** database (no pre-existing schema).
3. Row-count parity across every public table.
4. Order-independent md5 over every invoice's money columns
   (`id|total|tax|paid|status`).
5. Double-entry conservation on the **restored** ledger, globally and
   per tenant.

Planned v2: drive the in-product reconciler per tenant on the restored
instance for the full leg-completeness / abnormal-balance sweep.
