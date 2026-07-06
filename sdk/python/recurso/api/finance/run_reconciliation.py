from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.run_reconciliation_response_200 import RunReconciliationResponse200
from ...types import Response


def _get_kwargs() -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/finance/reconciliation",
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | RunReconciliationResponse200 | None:
    if response.status_code == 200:
        response_200 = RunReconciliationResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | RunReconciliationResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
) -> Response[Error | RunReconciliationResponse200]:
    """Run an on-demand ledger reconciliation

     Reconciles the tenant's invoices against its Postgres ledger entries and
    returns the computed drift report. Nothing is persisted. When TigerBeetle
    is connected its transfers are enumerated and compared against the
    PostgreSQL ledger (missing_in_tigerbeetle / missing_in_postgres /
    tb_amount_mismatch); otherwise `tb_compared` is false with the
    reason in `tb_skip_reason`.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | RunReconciliationResponse200]
    """

    kwargs = _get_kwargs()

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
) -> Error | RunReconciliationResponse200 | None:
    """Run an on-demand ledger reconciliation

     Reconciles the tenant's invoices against its Postgres ledger entries and
    returns the computed drift report. Nothing is persisted. When TigerBeetle
    is connected its transfers are enumerated and compared against the
    PostgreSQL ledger (missing_in_tigerbeetle / missing_in_postgres /
    tb_amount_mismatch); otherwise `tb_compared` is false with the
    reason in `tb_skip_reason`.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | RunReconciliationResponse200
    """

    return sync_detailed(
        client=client,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
) -> Response[Error | RunReconciliationResponse200]:
    """Run an on-demand ledger reconciliation

     Reconciles the tenant's invoices against its Postgres ledger entries and
    returns the computed drift report. Nothing is persisted. When TigerBeetle
    is connected its transfers are enumerated and compared against the
    PostgreSQL ledger (missing_in_tigerbeetle / missing_in_postgres /
    tb_amount_mismatch); otherwise `tb_compared` is false with the
    reason in `tb_skip_reason`.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | RunReconciliationResponse200]
    """

    kwargs = _get_kwargs()

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
) -> Error | RunReconciliationResponse200 | None:
    """Run an on-demand ledger reconciliation

     Reconciles the tenant's invoices against its Postgres ledger entries and
    returns the computed drift report. Nothing is persisted. When TigerBeetle
    is connected its transfers are enumerated and compared against the
    PostgreSQL ledger (missing_in_tigerbeetle / missing_in_postgres /
    tb_amount_mismatch); otherwise `tb_compared` is false with the
    reason in `tb_skip_reason`.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | RunReconciliationResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
        )
    ).parsed
