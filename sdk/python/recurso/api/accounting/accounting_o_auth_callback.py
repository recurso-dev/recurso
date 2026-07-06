from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.accounting_o_auth_callback_provider import AccountingOAuthCallbackProvider
from ...models.accounting_o_auth_callback_response_200 import AccountingOAuthCallbackResponse200
from ...models.error import Error
from ...types import UNSET, Response, Unset


def _get_kwargs(
    provider: AccountingOAuthCallbackProvider,
    *,
    code: str,
    state: str,
    realm_id: str | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["code"] = code

    params["state"] = state

    params["realmId"] = realm_id

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/accounting/callback/{provider}".format(
            provider=quote(str(provider), safe=""),
        ),
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> AccountingOAuthCallbackResponse200 | Error | None:
    if response.status_code == 200:
        response_200 = AccountingOAuthCallbackResponse200.from_dict(response.json())

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
) -> Response[AccountingOAuthCallbackResponse200 | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    provider: AccountingOAuthCallbackProvider,
    *,
    client: AuthenticatedClient | Client,
    code: str,
    state: str,
    realm_id: str | Unset = UNSET,
) -> Response[AccountingOAuthCallbackResponse200 | Error]:
    """OAuth callback for accounting providers

     Redirect target for QuickBooks/Xero. Exchanges the authorization code for tokens and stores (or
    refreshes) the connection.

    Args:
        provider (AccountingOAuthCallbackProvider):
        code (str):
        state (str):
        realm_id (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AccountingOAuthCallbackResponse200 | Error]
    """

    kwargs = _get_kwargs(
        provider=provider,
        code=code,
        state=state,
        realm_id=realm_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    provider: AccountingOAuthCallbackProvider,
    *,
    client: AuthenticatedClient | Client,
    code: str,
    state: str,
    realm_id: str | Unset = UNSET,
) -> AccountingOAuthCallbackResponse200 | Error | None:
    """OAuth callback for accounting providers

     Redirect target for QuickBooks/Xero. Exchanges the authorization code for tokens and stores (or
    refreshes) the connection.

    Args:
        provider (AccountingOAuthCallbackProvider):
        code (str):
        state (str):
        realm_id (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AccountingOAuthCallbackResponse200 | Error
    """

    return sync_detailed(
        provider=provider,
        client=client,
        code=code,
        state=state,
        realm_id=realm_id,
    ).parsed


async def asyncio_detailed(
    provider: AccountingOAuthCallbackProvider,
    *,
    client: AuthenticatedClient | Client,
    code: str,
    state: str,
    realm_id: str | Unset = UNSET,
) -> Response[AccountingOAuthCallbackResponse200 | Error]:
    """OAuth callback for accounting providers

     Redirect target for QuickBooks/Xero. Exchanges the authorization code for tokens and stores (or
    refreshes) the connection.

    Args:
        provider (AccountingOAuthCallbackProvider):
        code (str):
        state (str):
        realm_id (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AccountingOAuthCallbackResponse200 | Error]
    """

    kwargs = _get_kwargs(
        provider=provider,
        code=code,
        state=state,
        realm_id=realm_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    provider: AccountingOAuthCallbackProvider,
    *,
    client: AuthenticatedClient | Client,
    code: str,
    state: str,
    realm_id: str | Unset = UNSET,
) -> AccountingOAuthCallbackResponse200 | Error | None:
    """OAuth callback for accounting providers

     Redirect target for QuickBooks/Xero. Exchanges the authorization code for tokens and stores (or
    refreshes) the connection.

    Args:
        provider (AccountingOAuthCallbackProvider):
        code (str):
        state (str):
        realm_id (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AccountingOAuthCallbackResponse200 | Error
    """

    return (
        await asyncio_detailed(
            provider=provider,
            client=client,
            code=code,
            state=state,
            realm_id=realm_id,
        )
    ).parsed
