#!/usr/bin/env python3
"""No-network smoke test for the generated `recurso` Python SDK.

Verifies the package imports, the core resource endpoints exist with the
expected call signatures, request models serialize correctly, and the
authenticated client sends a Bearer token — all without making a single
HTTP request.

Run:  pip install ./sdk/python && python3 sdk/python/tests/smoke_test.py
"""

import inspect
import sys

FAILURES: list[str] = []
CHECKS_RUN = 0


def check(name: str, fn) -> None:
    global CHECKS_RUN
    CHECKS_RUN += 1
    try:
        fn()
        print(f"  ok  {name}")
    except Exception as exc:  # noqa: BLE001 - report every failure
        FAILURES.append(f"{name}: {exc!r}")
        print(f"FAIL  {name}: {exc!r}")


def params_of(func) -> list[str]:
    return list(inspect.signature(func).parameters)


def assert_endpoint(module, *, sync_params: list[str]) -> None:
    """Every generated endpoint module exposes sync/async call styles."""
    for style in ("sync", "sync_detailed", "asyncio", "asyncio_detailed"):
        assert hasattr(module, style), f"missing {style}()"
    got = params_of(module.sync)
    assert got == sync_params, f"sync() params {got} != {sync_params}"


def main() -> int:
    # --- package import and client construction -------------------------
    def import_package():
        import recurso

        assert recurso.Client is not None
        assert recurso.AuthenticatedClient is not None

    check("import recurso (Client, AuthenticatedClient exported)", import_package)

    def bearer_auth():
        from recurso import AuthenticatedClient

        client = AuthenticatedClient(base_url="http://localhost:8080", token="rsk_test_key")
        httpx_client = client.get_httpx_client()  # builds the client; no request sent
        assert httpx_client.headers["Authorization"] == "Bearer rsk_test_key"

    check("AuthenticatedClient sends Authorization: Bearer <token>", bearer_auth)

    # --- core resource endpoints (mirrors Node SDK v1 coverage) ---------
    def endpoints():
        from recurso.api.coupons import create_coupon, list_coupons
        from recurso.api.customers import create_customer, list_customers
        from recurso.api.entitlements import (
            check_entitlement,
            get_customer_entitlements,
            get_plan_entitlements,
            set_plan_entitlements,
        )
        from recurso.api.invoices import download_invoice_pdf, list_invoices
        from recurso.api.plans import create_plan, list_plans
        from recurso.api.subscriptions import (
            cancel_subscription,
            create_subscription,
            list_subscriptions,
            pause_subscription,
            resume_subscription,
        )
        from recurso.api.usage import record_usage_event

        assert_endpoint(create_plan, sync_params=["client", "body"])
        assert_endpoint(list_plans, sync_params=["client", "q", "limit", "page"])
        assert_endpoint(create_customer, sync_params=["client", "body"])
        assert_endpoint(list_customers, sync_params=["client", "q", "country", "status", "limit", "page"])
        assert_endpoint(create_subscription, sync_params=["client", "body"])
        assert_endpoint(list_subscriptions, sync_params=["client", "status", "q", "limit", "page"])
        assert_endpoint(cancel_subscription, sync_params=["id", "client", "body"])
        assert_endpoint(pause_subscription, sync_params=["id", "client"])
        assert_endpoint(resume_subscription, sync_params=["id", "client"])
        assert_endpoint(list_invoices, sync_params=["client"])
        assert_endpoint(create_coupon, sync_params=["client", "body"])
        assert_endpoint(list_coupons, sync_params=["client"])
        assert_endpoint(record_usage_event, sync_params=["client", "body"])
        assert_endpoint(check_entitlement, sync_params=["client", "customer_id", "feature"])
        assert_endpoint(get_plan_entitlements, sync_params=["id", "client"])
        assert_endpoint(set_plan_entitlements, sync_params=["id", "client", "body"])
        assert_endpoint(get_customer_entitlements, sync_params=["id", "client"])
        # PDF download exists as a real endpoint (Node exposes pdfUrl()).
        assert hasattr(download_invoice_pdf, "sync_detailed")

    check("core endpoints exist with expected signatures", endpoints)

    # --- delivery tracking, redelivery, and FX-normalized MRR ------------
    def delivery_and_mrr_endpoints():
        from recurso.api.analytics import get_mrr
        from recurso.api.webhooks import (
            list_event_deliveries,
            list_webhook_endpoint_deliveries,
            redeliver_event,
        )

        assert_endpoint(list_event_deliveries, sync_params=["id", "client"])
        assert_endpoint(
            list_webhook_endpoint_deliveries,
            sync_params=["id", "client", "limit", "offset", "status"],
        )
        assert_endpoint(redeliver_event, sync_params=["id", "client"])
        assert_endpoint(get_mrr, sync_params=["client"])

    check("delivery tracking + MRR endpoints exist with expected signatures", delivery_and_mrr_endpoints)

    def redeliver_kwargs():
        from uuid import UUID

        from recurso.api.webhooks import redeliver_event

        kwargs = redeliver_event._get_kwargs(id=UUID("00000000-0000-0000-0000-000000000003"))
        assert kwargs["method"] == "post"
        assert kwargs["url"] == "/v1/events/00000000-0000-0000-0000-000000000003/redeliver"

    check("redeliver_event builds POST /v1/events/{id}/redeliver", redeliver_kwargs)

    def fx_models():
        from recurso.models import EventDeliveryStatus, FXSnapshotSource, MRRMetrics

        # FX provenance is part of the MRR response contract.
        fields = {f.name for f in MRRMetrics.__attrs_attrs__}
        for expected in ("mrr", "normalized_mrr", "reporting_currency", "breakdown", "fx"):
            assert expected in fields, f"MRRMetrics missing {expected!r} (has {sorted(fields)})"
        assert {s.value for s in FXSnapshotSource} >= {"live", "static-fallback"}
        assert {s.value for s in EventDeliveryStatus} == {"pending", "succeeded", "failed"}

    check("MRRMetrics carries FX provenance; delivery status enum matches API", fx_models)

    # --- request models serialize as the API expects --------------------
    def plan_model():
        from recurso.models import CreatePlanRequest, CreatePlanRequestIntervalUnit

        body = CreatePlanRequest(
            name="Pro Plan",
            code="PRO-USD",
            amount=2900,
            currency="USD",
            interval_unit=CreatePlanRequestIntervalUnit.MONTH,
            interval_count=1,
        ).to_dict()
        assert body == {
            "name": "Pro Plan",
            "code": "PRO-USD",
            "amount": 2900,
            "currency": "USD",
            "interval_unit": "month",
            "interval_count": 1,
        }, body

    check("CreatePlanRequest.to_dict() round-trips", plan_model)

    def customer_model():
        from recurso.models import CreateCustomerRequest

        body = CreateCustomerRequest(name="Jane User", email="jane@example.com", country="US").to_dict()
        assert body["name"] == "Jane User"
        assert body["email"] == "jane@example.com"
        assert body["country"] == "US"
        # Unset optional address fields must not be serialized.
        assert "line1" not in body, body

    check("CreateCustomerRequest omits unset optionals", customer_model)

    def subscription_model():
        from uuid import UUID

        from recurso.models import CreateSubscriptionRequest

        body = CreateSubscriptionRequest(
            customer_id=UUID("00000000-0000-0000-0000-000000000001"),
            plan_id=UUID("00000000-0000-0000-0000-000000000002"),
        ).to_dict()
        assert body["customer_id"].endswith("0001")
        assert body["plan_id"].endswith("0002")

    check("CreateSubscriptionRequest serializes UUIDs", subscription_model)

    # --- request construction stays offline ------------------------------
    def kwargs_only():
        from uuid import UUID

        from recurso.api.entitlements import check_entitlement

        kwargs = check_entitlement._get_kwargs(
            customer_id=UUID("00000000-0000-0000-0000-000000000001"),
            feature="sso",
        )
        assert kwargs["method"] == "get"
        assert kwargs["url"] == "/v1/entitlements/check"
        assert kwargs["params"]["feature"] == "sso"

    check("check_entitlement builds GET /v1/entitlements/check", kwargs_only)

    print()
    if FAILURES:
        print(f"smoke test FAILED ({len(FAILURES)} of {CHECKS_RUN} check(s))")
        return 1
    print(f"smoke test passed ({CHECKS_RUN} checks)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
