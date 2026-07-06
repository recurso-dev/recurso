# recurso

Official Python SDK for the [Recurso](https://github.com/swapnull-in/recur-so) billing API.

Generated from the API's OpenAPI 3.1 specification (`cmd/api/openapi.yaml`) with
[openapi-python-client](https://github.com/openapi-generators/openapi-python-client), so it covers the
full REST surface — plans, customers, the subscription lifecycle, invoices, usage events, coupons,
quotes, entitlements, webhooks, and more — with typed request/response models and both sync and async
call styles.

Requires Python 3.11+.

> Not yet published to PyPI. Install from the repository for now:

```bash
pip install ./sdk/python          # from the repo root
# or, for development:
pip install -e ./sdk/python
```

## Quickstart

Mirrors the Node SDK quickstart: create a plan, a customer, and a subscription, then gate a feature
with an entitlement check.

```python
from recurso import AuthenticatedClient
from recurso.api.plans import create_plan
from recurso.api.customers import create_customer
from recurso.api.subscriptions import create_subscription
from recurso.api.entitlements import check_entitlement
from recurso.models import (
    CreatePlanRequest,
    CreatePlanRequestIntervalUnit,
    CreateCustomerRequest,
    CreateSubscriptionRequest,
)

client = AuthenticatedClient(
    base_url="https://billing.example.com",
    token="rsk_live_your_api_key",   # sent as: Authorization: Bearer <token>
)

with client:
    plan = create_plan.sync(
        client=client,
        body=CreatePlanRequest(
            name="Pro Plan",
            code="PRO-USD",
            amount=2900,  # minor units (cents)
            currency="USD",
            interval_unit=CreatePlanRequestIntervalUnit.MONTH,
            interval_count=1,
        ),
    )

    customer = create_customer.sync(
        client=client,
        body=CreateCustomerRequest(
            name="Jane User",
            email="jane@example.com",
            country="US",
        ),
    )

    subscription = create_subscription.sync(
        client=client,
        body=CreateSubscriptionRequest(
            customer_id=customer.id,
            plan_id=plan.id,
        ),
    )

    # Fast single-feature entitlement check (the feature-gating hot path).
    result = check_entitlement.sync(
        client=client,
        customer_id=customer.id,
        feature="sso",
    )
```

## Call styles

Every endpoint module exposes four functions:

1. `sync`: blocking request, returns the parsed body (or `None` on error statuses)
2. `sync_detailed`: blocking request, returns a `Response` with `status_code`, `headers`, and `parsed`
3. `asyncio`: like `sync`, but awaitable
4. `asyncio_detailed`: like `sync_detailed`, but awaitable

```python
import asyncio
from recurso.api.plans import list_plans

async def main():
    async with client:
        plans = await list_plans.asyncio(client=client)

asyncio.run(main())
```

To raise `recurso.errors.UnexpectedStatus` on undocumented status codes instead of returning `None`,
construct the client with `raise_on_unexpected_status=True`.

## Advanced

The client accepts standard `httpx` options (`timeout`, `verify_ssl`, `headers`, `cookies`,
`httpx_args`) and supports `.with_headers()` / `.with_timeout()` for per-call variants. See
`recurso/client.py` for the full surface. Mutating endpoints support idempotency via the
`X-Idempotency-Key` header:

```python
client = client.with_headers({"X-Idempotency-Key": "order-1234"})
```

## Regenerating

This package is generated — do not hand-edit files under `recurso/`. To regenerate after a spec
change (from the repo root):

```bash
pipx run openapi-python-client generate \
  --path cmd/api/openapi.yaml \
  --config <(printf 'project_name_override: recurso\npackage_name_override: recurso\n') \
  --output-path sdk/python \
  --overwrite
```

Note: the generator emits one benign warning for `GET /openapi.yaml` (the spec self-serve endpoint)
because its `application/yaml` response body is not a JSON media type; that response is simply
omitted from the generated client.

## Testing

A no-network smoke test verifies the package imports and key endpoint signatures exist:

```bash
pip install ./sdk/python
python3 sdk/python/tests/smoke_test.py
```

## License

MIT
