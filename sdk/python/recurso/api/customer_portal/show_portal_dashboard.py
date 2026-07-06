from http import HTTPStatus
from typing import Any
from urllib.parse import quote
from uuid import UUID

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...types import Response


def _get_kwargs(
    customer_id: UUID,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/portal/{customer_id}".format(
            customer_id=quote(str(customer_id), safe=""),
        ),
    }

    return _kwargs


def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> str | None:
    if response.status_code == 200:
        response_200 = response.text
        return response_200

    if response.status_code == 400:
        response_400 = response.text
        return response_400

    if response.status_code == 404:
        response_404 = response.text
        return response_404

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[str]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    customer_id: UUID,
    *,
    client: AuthenticatedClient | Client,
) -> Response[str]:
    """Legacy HTML customer dashboard

     Server-rendered HTML dashboard listing the customer's invoices. Public, rate-limited.

    Args:
        customer_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[str]
    """

    kwargs = _get_kwargs(
        customer_id=customer_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    customer_id: UUID,
    *,
    client: AuthenticatedClient | Client,
) -> str | None:
    """Legacy HTML customer dashboard

     Server-rendered HTML dashboard listing the customer's invoices. Public, rate-limited.

    Args:
        customer_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        str
    """

    return sync_detailed(
        customer_id=customer_id,
        client=client,
    ).parsed


async def asyncio_detailed(
    customer_id: UUID,
    *,
    client: AuthenticatedClient | Client,
) -> Response[str]:
    """Legacy HTML customer dashboard

     Server-rendered HTML dashboard listing the customer's invoices. Public, rate-limited.

    Args:
        customer_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[str]
    """

    kwargs = _get_kwargs(
        customer_id=customer_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    customer_id: UUID,
    *,
    client: AuthenticatedClient | Client,
) -> str | None:
    """Legacy HTML customer dashboard

     Server-rendered HTML dashboard listing the customer's invoices. Public, rate-limited.

    Args:
        customer_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        str
    """

    return (
        await asyncio_detailed(
            customer_id=customer_id,
            client=client,
        )
    ).parsed
