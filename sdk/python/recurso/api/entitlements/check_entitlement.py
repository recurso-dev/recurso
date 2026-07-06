from http import HTTPStatus
from typing import Any
from uuid import UUID

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.check_entitlement_response_200 import CheckEntitlementResponse200
from ...models.error import Error
from ...types import UNSET, Response


def _get_kwargs(
    *,
    customer_id: UUID,
    feature: str,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    json_customer_id = str(customer_id)
    params["customer_id"] = json_customer_id

    params["feature"] = feature

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/entitlements/check",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> CheckEntitlementResponse200 | Error | None:
    if response.status_code == 200:
        response_200 = CheckEntitlementResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[CheckEntitlementResponse200 | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    customer_id: UUID,
    feature: str,
) -> Response[CheckEntitlementResponse200 | Error]:
    """Fast single-feature entitlement check

     The hot path for feature gating — one indexed query.

    Args:
        customer_id (UUID):
        feature (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CheckEntitlementResponse200 | Error]
    """

    kwargs = _get_kwargs(
        customer_id=customer_id,
        feature=feature,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    customer_id: UUID,
    feature: str,
) -> CheckEntitlementResponse200 | Error | None:
    """Fast single-feature entitlement check

     The hot path for feature gating — one indexed query.

    Args:
        customer_id (UUID):
        feature (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CheckEntitlementResponse200 | Error
    """

    return sync_detailed(
        client=client,
        customer_id=customer_id,
        feature=feature,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    customer_id: UUID,
    feature: str,
) -> Response[CheckEntitlementResponse200 | Error]:
    """Fast single-feature entitlement check

     The hot path for feature gating — one indexed query.

    Args:
        customer_id (UUID):
        feature (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CheckEntitlementResponse200 | Error]
    """

    kwargs = _get_kwargs(
        customer_id=customer_id,
        feature=feature,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    customer_id: UUID,
    feature: str,
) -> CheckEntitlementResponse200 | Error | None:
    """Fast single-feature entitlement check

     The hot path for feature gating — one indexed query.

    Args:
        customer_id (UUID):
        feature (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CheckEntitlementResponse200 | Error
    """

    return (
        await asyncio_detailed(
            client=client,
            customer_id=customer_id,
            feature=feature,
        )
    ).parsed
