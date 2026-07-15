from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.get_auth_oauth_provider_callback_provider import GetAuthOauthProviderCallbackProvider
from ...types import UNSET, Response, Unset


def _get_kwargs(
    provider: GetAuthOauthProviderCallbackProvider,
    *,
    code: str | Unset = UNSET,
    state: str | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["code"] = code

    params["state"] = state

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/auth/oauth/{provider}/callback".format(
            provider=quote(str(provider), safe=""),
        ),
        "params": params,
    }

    return _kwargs


def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | Error | None:
    if response.status_code == 302:
        response_302 = cast(Any, None)
        return response_302

    if response.status_code == 403:
        response_403 = Error.from_dict(response.json())

        return response_403

    if response.status_code == 404:
        response_404 = Error.from_dict(response.json())

        return response_404

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Any | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    provider: GetAuthOauthProviderCallbackProvider,
    *,
    client: AuthenticatedClient | Client,
    code: str | Unset = UNSET,
    state: str | Unset = UNSET,
) -> Response[Any | Error]:
    """OAuth callback (provider redirects here)

     Validates `state` against the cookie (constant-time), exchanges the code with PKCE, fetches userinfo
    and requires a verified email (Google: email_verified==true; GitHub: a primary verified email). Then
    find-or-create: (1) an existing identity logs in; (2) a matching verified email links a new identity
    and logs in; (3) a brand-new email creates a tenant + owner user. On success sets the
    `recurso_session` cookie and 302s to `{DASHBOARD_URL}/`. On failure 302s to
    `{DASHBOARD_URL}/login?error=oauth` (never an open redirect). A state mismatch returns 403.

    Args:
        provider (GetAuthOauthProviderCallbackProvider):
        code (str | Unset):
        state (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
    """

    kwargs = _get_kwargs(
        provider=provider,
        code=code,
        state=state,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    provider: GetAuthOauthProviderCallbackProvider,
    *,
    client: AuthenticatedClient | Client,
    code: str | Unset = UNSET,
    state: str | Unset = UNSET,
) -> Any | Error | None:
    """OAuth callback (provider redirects here)

     Validates `state` against the cookie (constant-time), exchanges the code with PKCE, fetches userinfo
    and requires a verified email (Google: email_verified==true; GitHub: a primary verified email). Then
    find-or-create: (1) an existing identity logs in; (2) a matching verified email links a new identity
    and logs in; (3) a brand-new email creates a tenant + owner user. On success sets the
    `recurso_session` cookie and 302s to `{DASHBOARD_URL}/`. On failure 302s to
    `{DASHBOARD_URL}/login?error=oauth` (never an open redirect). A state mismatch returns 403.

    Args:
        provider (GetAuthOauthProviderCallbackProvider):
        code (str | Unset):
        state (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
    """

    return sync_detailed(
        provider=provider,
        client=client,
        code=code,
        state=state,
    ).parsed


async def asyncio_detailed(
    provider: GetAuthOauthProviderCallbackProvider,
    *,
    client: AuthenticatedClient | Client,
    code: str | Unset = UNSET,
    state: str | Unset = UNSET,
) -> Response[Any | Error]:
    """OAuth callback (provider redirects here)

     Validates `state` against the cookie (constant-time), exchanges the code with PKCE, fetches userinfo
    and requires a verified email (Google: email_verified==true; GitHub: a primary verified email). Then
    find-or-create: (1) an existing identity logs in; (2) a matching verified email links a new identity
    and logs in; (3) a brand-new email creates a tenant + owner user. On success sets the
    `recurso_session` cookie and 302s to `{DASHBOARD_URL}/`. On failure 302s to
    `{DASHBOARD_URL}/login?error=oauth` (never an open redirect). A state mismatch returns 403.

    Args:
        provider (GetAuthOauthProviderCallbackProvider):
        code (str | Unset):
        state (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
    """

    kwargs = _get_kwargs(
        provider=provider,
        code=code,
        state=state,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    provider: GetAuthOauthProviderCallbackProvider,
    *,
    client: AuthenticatedClient | Client,
    code: str | Unset = UNSET,
    state: str | Unset = UNSET,
) -> Any | Error | None:
    """OAuth callback (provider redirects here)

     Validates `state` against the cookie (constant-time), exchanges the code with PKCE, fetches userinfo
    and requires a verified email (Google: email_verified==true; GitHub: a primary verified email). Then
    find-or-create: (1) an existing identity logs in; (2) a matching verified email links a new identity
    and logs in; (3) a brand-new email creates a tenant + owner user. On success sets the
    `recurso_session` cookie and 302s to `{DASHBOARD_URL}/`. On failure 302s to
    `{DASHBOARD_URL}/login?error=oauth` (never an open redirect). A state mismatch returns 403.

    Args:
        provider (GetAuthOauthProviderCallbackProvider):
        code (str | Unset):
        state (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
    """

    return (
        await asyncio_detailed(
            provider=provider,
            client=client,
            code=code,
            state=state,
        )
    ).parsed
