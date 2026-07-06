from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.create_virtual_account_body import CreateVirtualAccountBody
from ...models.error import Error
from ...models.virtual_account import VirtualAccount
from ...types import Response


def _get_kwargs(
    *,
    body: CreateVirtualAccountBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/virtual-accounts",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VirtualAccount | None:
    if response.status_code == 201:
        response_201 = VirtualAccount.from_dict(response.json())

        return response_201

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
) -> Response[Error | VirtualAccount]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: CreateVirtualAccountBody,
) -> Response[Error | VirtualAccount]:
    """Create a virtual account

     Provisions a virtual bank account (via Razorpay) that the customer can wire money to.

    Args:
        body (CreateVirtualAccountBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VirtualAccount]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    body: CreateVirtualAccountBody,
) -> Error | VirtualAccount | None:
    """Create a virtual account

     Provisions a virtual bank account (via Razorpay) that the customer can wire money to.

    Args:
        body (CreateVirtualAccountBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VirtualAccount
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: CreateVirtualAccountBody,
) -> Response[Error | VirtualAccount]:
    """Create a virtual account

     Provisions a virtual bank account (via Razorpay) that the customer can wire money to.

    Args:
        body (CreateVirtualAccountBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VirtualAccount]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: CreateVirtualAccountBody,
) -> Error | VirtualAccount | None:
    """Create a virtual account

     Provisions a virtual bank account (via Razorpay) that the customer can wire money to.

    Args:
        body (CreateVirtualAccountBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VirtualAccount
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
