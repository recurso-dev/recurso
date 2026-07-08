# Recurso — Postman collection

A ready-to-use Postman collection for the Recurso API, generated from the
[OpenAPI spec](../cmd/api/openapi.yaml). It covers every endpoint, grouped by
resource.

## Import

1. In Postman: **Import** → drag in both files:
   - `recurso.postman_collection.json` — the requests
   - `recurso.postman_environment.json` — the variables
2. Select the **Recurso** environment (top-right).
3. Set the environment variables:
   - `baseUrl` — your API host (default `https://api.recurso.dev`; for local dev use
     `http://localhost:8080/v1`)
   - `bearerToken` — a secret API key from **Developers** in the dashboard (the local
     demo key is `sk_test_12345`)

Every request authenticates automatically via the collection's bearer auth
(`Authorization: Bearer {{bearerToken}}`).

## Regenerating

The collection is generated from the OpenAPI spec — regenerate after API changes:

```bash
npx openapi-to-postmanv2 \
  -s cmd/api/openapi.yaml \
  -o postman/recurso.postman_collection.json \
  -p -O folderStrategy=Tags,requestParametersResolution=Example
```

Then re-set the default `baseUrl` (the converter resets it to the spec's server URL).
