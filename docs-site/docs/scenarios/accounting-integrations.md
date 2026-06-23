---
sidebar_position: 11
---

# Accounting Integrations

Recurso integrates with accounting platforms like QuickBooks and Xero to keep your financial records in sync. Customers and invoices created in Recurso are automatically pushed to your connected accounting software. The integration uses OAuth 2.0 for secure authorization and includes a background worker that runs daily to sync data and refresh tokens.

## How the Sync Works

1. **Connect**: You initiate an OAuth flow with your accounting provider (QuickBooks or Xero). Recurso stores the access and refresh tokens securely.
2. **Automatic sync**: A background worker (`AccountingSyncWorker`) runs every 24 hours. For each active connection, it syncs customers and invoices to the accounting platform.
3. **Token refresh**: Before each sync, the worker checks if the OAuth access token is expired. If so, it uses the refresh token to obtain a new access token automatically.
4. **Manual sync**: You can trigger a sync on demand at any time via the API.

## Managing Connections

### List Connections

**GET** `/v1/accounting/connections`

Returns all accounting connections for your tenant.

```bash
curl https://api.recurso.dev/v1/accounting/connections \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "data": [
    {
      "id": "c1d2e3f4-a5b6-7890-cdef-1234567890ab",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "provider": "quickbooks",
      "realm_id": "4620816365181753230",
      "sync_status": "idle",
      "is_active": true,
      "last_synced_at": "2026-06-22T03:00:00Z",
      "created_at": "2026-06-01T10:00:00Z"
    }
  ]
}
```

### Initiate OAuth Connection

**POST** `/v1/accounting/connect/:provider`

Starts the OAuth 2.0 authorization flow. The `provider` parameter must be either `quickbooks` or `xero`. The response contains the `auth_url` that you should redirect the user to in a browser.

```bash
curl -X POST https://api.recurso.dev/v1/accounting/connect/quickbooks \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "auth_url": "https://appcenter.intuit.com/connect/oauth2?client_id=ABc123...&redirect_uri=https%3A%2F%2Fapi.recurso.dev%2Fv1%2Faccounting%2Fcallback%2Fquickbooks&scope=com.intuit.quickbooks.accounting&state=f47ac10b-58cc-4372-a567-0e02b2c3d479%3Aquickbooks%3Aabcdef1234..."
}
```

For Xero:

```bash
curl -X POST https://api.recurso.dev/v1/accounting/connect/xero \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "auth_url": "https://login.xero.com/identity/connect/authorize?client_id=XYz789...&redirect_uri=https%3A%2F%2Fapi.recurso.dev%2Fv1%2Faccounting%2Fcallback%2Fxero&scope=openid+profile+email+accounting.transactions+accounting.contacts&state=f47ac10b-58cc-4372-a567-0e02b2c3d479%3Axero%3Afedcba5678..."
}
```

| Path Parameter | Type   | Required | Description                              |
|----------------|--------|----------|------------------------------------------|
| `provider`     | string | Yes      | `quickbooks` or `xero`.                  |

### OAuth Callback

**GET** `/v1/accounting/callback/:provider`

This endpoint is called by the accounting provider after the user authorizes access. Recurso exchanges the authorization code for access and refresh tokens, then stores the connection.

You typically do not call this endpoint directly -- it is the OAuth redirect URI. However, it is documented here for completeness.

```
GET https://api.recurso.dev/v1/accounting/callback/quickbooks?code=AB11234...&state=f47ac10b...&realmId=4620816365181753230
```

Response (`200 OK`):

```json
{
  "status": "connected",
  "connection_id": "c1d2e3f4-a5b6-7890-cdef-1234567890ab"
}
```

The `state` parameter is HMAC-signed to prevent CSRF attacks. Recurso verifies the signature before processing the callback.

### Disconnect

**DELETE** `/v1/accounting/connections/:id`

Removes the accounting connection. This revokes the stored tokens and stops syncing.

```bash
curl -X DELETE https://api.recurso.dev/v1/accounting/connections/c1d2e3f4-a5b6-7890-cdef-1234567890ab \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "status": "disconnected"
}
```

## Syncing Data

### Trigger Manual Sync

**POST** `/v1/accounting/sync`

Triggers an immediate sync of customers and invoices to all active accounting connections for your tenant.

```bash
curl -X POST https://api.recurso.dev/v1/accounting/sync \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "status": "sync_triggered"
}
```

The sync runs asynchronously. Use the sync status endpoint to monitor progress.

### Check Sync Status

**GET** `/v1/accounting/sync/status`

Returns the most recent sync logs for your tenant (up to 50 entries).

```bash
curl https://api.recurso.dev/v1/accounting/sync/status \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "data": [
    {
      "id": "d2e3f4a5-b6c7-8901-def0-1234567890ab",
      "connection_id": "c1d2e3f4-a5b6-7890-cdef-1234567890ab",
      "provider": "quickbooks",
      "status": "completed",
      "customers_synced": 47,
      "invoices_synced": 132,
      "errors": [],
      "started_at": "2026-06-23T03:00:00Z",
      "completed_at": "2026-06-23T03:02:14Z"
    },
    {
      "id": "e3f4a5b6-c7d8-9012-ef01-234567890abc",
      "connection_id": "c1d2e3f4-a5b6-7890-cdef-1234567890ab",
      "provider": "quickbooks",
      "status": "completed",
      "customers_synced": 45,
      "invoices_synced": 128,
      "errors": [],
      "started_at": "2026-06-22T03:00:00Z",
      "completed_at": "2026-06-22T03:01:58Z"
    }
  ]
}
```

## Configuration

### Environment Variables

To use accounting integrations, you need to set the following environment variables for your chosen provider:

**QuickBooks:**

| Variable            | Description                          |
|---------------------|--------------------------------------|
| `QBO_CLIENT_ID`     | QuickBooks OAuth client ID.          |
| `QBO_CLIENT_SECRET` | QuickBooks OAuth client secret.      |

**Xero:**

| Variable             | Description                    |
|----------------------|--------------------------------|
| `XERO_CLIENT_ID`     | Xero OAuth client ID.          |
| `XERO_CLIENT_SECRET` | Xero OAuth client secret.      |

**General:**

| Variable             | Description                                              |
|----------------------|----------------------------------------------------------|
| `APP_BASE_URL`       | The base URL of your Recurso instance (used for OAuth redirect URIs). Defaults to `http://localhost:8080`. |
| `OAUTH_STATE_SECRET` | Secret key used to HMAC-sign OAuth state parameters. If not set, a fallback key is used (not recommended for production). |

## What Gets Synced

| Recurso Entity | Accounting Platform Record | Direction       |
|----------------|---------------------------|-----------------|
| Customer       | Customer / Contact        | Recurso -> Provider |
| Invoice        | Invoice / Bill            | Recurso -> Provider |

The sync is one-directional: data flows from Recurso to the accounting platform. Changes made directly in QuickBooks or Xero are not synced back to Recurso.

## Example: End-to-End QuickBooks Setup

**Step 1**: Configure your environment with QuickBooks credentials.

```bash
export QBO_CLIENT_ID="ABcDeFgHiJkLmNoPqRsTuVwXyZ"
export QBO_CLIENT_SECRET="aBcDeFgHiJkLmNoPqRsTuV"
export APP_BASE_URL="https://api.recurso.dev"
export OAUTH_STATE_SECRET="my-secure-random-secret-key"
```

**Step 2**: Initiate the OAuth flow.

```bash
curl -X POST https://api.recurso.dev/v1/accounting/connect/quickbooks \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

**Step 3**: Redirect the user to the returned `auth_url`. After they authorize, Recurso automatically handles the callback, exchanges the code for tokens, and stores the connection.

**Step 4**: Verify the connection.

```bash
curl https://api.recurso.dev/v1/accounting/connections \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

**Step 5**: The daily worker will sync automatically. To trigger an immediate sync:

```bash
curl -X POST https://api.recurso.dev/v1/accounting/sync \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```
