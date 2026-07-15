# recurso-node

Official Node.js SDK for the [Recurso](https://github.com/recurso-dev/recurso) billing API — 17 resources, 64 methods covering plans, customers, the full subscription lifecycle, invoices, usage events, quotes, entitlements, webhooks (including delivery tracking and redelivery), analytics, and more. Every method is covered by the vitest suite in `test/` (`npm test`).

> Not yet published to npm. Install from the repository for now:

```bash
cd sdk/node
npm install && npm run build
npm link   # or npm pack and install the tarball
```

## Usage

```typescript
import { Recurso } from 'recurso-node';

const recurso = new Recurso('sk_live_your_api_key', 'https://billing.example.com');

const plan = await recurso.plans.create({
  name: 'Pro Plan',
  code: 'PRO-USD',
  amount: 2900,          // minor units
  currency: 'USD',
  interval_unit: 'month',
});

const customer = await recurso.customers.create({
  name: 'Jane User',
  email: 'jane@example.com',
  country: 'US',
});

await recurso.subscriptions.create({
  customer_id: customer.id,
  plan_id: plan.id,
});
```

Full method reference and guides: [docs.recurso.dev](https://docs.recurso.dev).

## Typed responses

Requests and responses are fully typed from the API's OpenAPI spec, so results
carry concrete field types (not an opaque object) and editors autocomplete them:

```typescript
const { data } = await recurso.subscriptions.list();
data?.forEach((s) => console.log(s.id, s.status)); // s is Subscription

const sub = await recurso.subscriptions.create({
  customer_id: customer.id!,
  plan_id: plan.id!,
});
sub.current_period_end; // string | undefined — typed, autocompleted
```

Resource types are exported for annotating your own code:

```typescript
import type { Subscription, Customer, Invoice } from 'recurso-node';
```

### Keeping types in sync (maintainers)

`src/schema.d.ts` is generated from the server's OpenAPI spec — the same source
of truth the Python SDK is generated from — so the SDK types can never drift
from the API. After changing the API, regenerate and verify:

```bash
npm run generate    # regenerate src/schema.d.ts from ../../cmd/api/openapi.yaml
npm run typecheck   # tsc over the SDK + response-typing assertions in test/
npm test            # vitest suite
```

## License

MIT
