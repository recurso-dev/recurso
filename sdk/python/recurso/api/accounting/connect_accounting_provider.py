from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.connect_accounting_provider_provider import ConnectAccountingProviderProvider
from ...models.connect_accounting_provider_response_200 import ConnectAccountingProviderResponse200
from ...models.error import Error
from ...types import Response


def _get_kwargs(
    provider: ConnectAccountingProviderProvider,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/accounting/connect/{provider}".format(
            provider=quote(str(provider), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ConnectAccountingProviderResponse200 | Error | None:
    if response.status_code == 200:
        response_200 = ConnectAccountingProviderResponse200.from_dict(response.json())

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
) -> Response[ConnectAccountingProviderResponse200 | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    provider: ConnectAccountingProviderProvider,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ConnectAccountingProviderResponse200 | Error]:
    """Start an accounting OAuth flow

     Returns the provider authorization URL. The state parameter is HMAC-signed with the tenant identity.

    Args:
        provider (ConnectAccountingProviderProvider):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ConnectAccountingProviderResponse200 | Error]
    """

    kwargs = _get_kwargs(
        provider=provider,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    provider: ConnectAccountingProviderProvider,
    *,
    client: AuthenticatedClient | Client,
) -> ConnectAccountingProviderResponse200 | Error | None:
    """Start an accounting OAuth flow

     Returns the provider authorization URL. The state parameter is HMAC-signed with the tenant identity.

    Args:
        provider (ConnectAccountingProviderProvider):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ConnectAccountingProviderResponse200 | Error
    """

    return sync_detailed(
        provider=provider,
        client=client,
    ).parsed


async def asyncio_detailed(
    provider: ConnectAccountingProviderProvider,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ConnectAccountingProviderResponse200 | Error]:
    """Start an accounting OAuth flow

     Returns the provider authorization URL. The state parameter is HMAC-signed with the tenant identity.

    Args:
        provider (ConnectAccountingProviderProvider):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ConnectAccountingProviderResponse200 | Error]
    """

    kwargs = _get_kwargs(
        provider=provider,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    provider: ConnectAccountingProviderProvider,
    *,
    client: AuthenticatedClient | Client,
) -> ConnectAccountingProviderResponse200 | Error | None:
    """Start an accounting OAuth flow

     Returns the provider authorization URL. The state parameter is HMAC-signed with the tenant identity.

    Args:
        provider (ConnectAccountingProviderProvider):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ConnectAccountingProviderResponse200 | Error
    """

    return (
        await asyncio_detailed(
            provider=provider,
            client=client,
        )
    ).parsed
