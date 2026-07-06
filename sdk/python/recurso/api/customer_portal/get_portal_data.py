from http import HTTPStatus
from typing import Any
from urllib.parse import quote
from uuid import UUID

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.get_portal_data_response_200 import GetPortalDataResponse200
from ...types import Response


def _get_kwargs(
    tenant_id: UUID,
    customer_id: UUID,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/portal/{tenant_id}/{customer_id}".format(
            tenant_id=quote(str(tenant_id), safe=""),
            customer_id=quote(str(customer_id), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | GetPortalDataResponse200 | None:
    if response.status_code == 200:
        response_200 = GetPortalDataResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 404:
        response_404 = Error.from_dict(response.json())

        return response_404

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | GetPortalDataResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    tenant_id: UUID,
    customer_id: UUID,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Error | GetPortalDataResponse200]:
    """Read-only portal data (JSON)

     Unauthenticated, rate-limited JSON payload consumed by the React portal — customer profile,
    subscriptions, and invoices.

    Args:
        tenant_id (UUID):
        customer_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetPortalDataResponse200]
    """

    kwargs = _get_kwargs(
        tenant_id=tenant_id,
        customer_id=customer_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    tenant_id: UUID,
    customer_id: UUID,
    *,
    client: AuthenticatedClient | Client,
) -> Error | GetPortalDataResponse200 | None:
    """Read-only portal data (JSON)

     Unauthenticated, rate-limited JSON payload consumed by the React portal — customer profile,
    subscriptions, and invoices.

    Args:
        tenant_id (UUID):
        customer_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetPortalDataResponse200
    """

    return sync_detailed(
        tenant_id=tenant_id,
        customer_id=customer_id,
        client=client,
    ).parsed


async def asyncio_detailed(
    tenant_id: UUID,
    customer_id: UUID,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Error | GetPortalDataResponse200]:
    """Read-only portal data (JSON)

     Unauthenticated, rate-limited JSON payload consumed by the React portal — customer profile,
    subscriptions, and invoices.

    Args:
        tenant_id (UUID):
        customer_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetPortalDataResponse200]
    """

    kwargs = _get_kwargs(
        tenant_id=tenant_id,
        customer_id=customer_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    tenant_id: UUID,
    customer_id: UUID,
    *,
    client: AuthenticatedClient | Client,
) -> Error | GetPortalDataResponse200 | None:
    """Read-only portal data (JSON)

     Unauthenticated, rate-limited JSON payload consumed by the React portal — customer profile,
    subscriptions, and invoices.

    Args:
        tenant_id (UUID):
        customer_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetPortalDataResponse200
    """

    return (
        await asyncio_detailed(
            tenant_id=tenant_id,
            customer_id=customer_id,
            client=client,
        )
    ).parsed
