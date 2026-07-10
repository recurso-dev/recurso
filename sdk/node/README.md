# recurso-node

Official Node.js SDK for the [Recurso](https://github.com/swapnull-in/recur-so) billing API — 17 resources, 64 methods covering plans, customers, the full subscription lifecycle, invoices, usage events, quotes, entitlements, webhooks (including delivery tracking and redelivery), analytics, and more. Every method is covered by the vitest suite in `test/` (`npm test`).

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

## License

MIT
