from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.get_auth_oauth_provider_start_provider import GetAuthOauthProviderStartProvider
from ...types import Response


def _get_kwargs(
    provider: GetAuthOauthProviderStartProvider,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/auth/oauth/{provider}/start".format(
            provider=quote(str(provider), safe=""),
        ),
    }

    return _kwargs


def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | Error | None:
    if response.status_code == 302:
        response_302 = cast(Any, None)
        return response_302

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
    provider: GetAuthOauthProviderStartProvider,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Any | Error]:
    """Begin an OAuth login (redirect to the provider)

     Generates a CSRF `state` and a PKCE verifier, binds them into a short-lived signed httpOnly cookie
    (`recurso_oauth_state`, scoped to /auth/oauth), and 302-redirects to the provider's authorize URL.
    Unknown or disabled providers return 404.

    Args:
        provider (GetAuthOauthProviderStartProvider):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
    """

    kwargs = _get_kwargs(
        provider=provider,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    provider: GetAuthOauthProviderStartProvider,
    *,
    client: AuthenticatedClient | Client,
) -> Any | Error | None:
    """Begin an OAuth login (redirect to the provider)

     Generates a CSRF `state` and a PKCE verifier, binds them into a short-lived signed httpOnly cookie
    (`recurso_oauth_state`, scoped to /auth/oauth), and 302-redirects to the provider's authorize URL.
    Unknown or disabled providers return 404.

    Args:
        provider (GetAuthOauthProviderStartProvider):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
    """

    return sync_detailed(
        provider=provider,
        client=client,
    ).parsed


async def asyncio_detailed(
    provider: GetAuthOauthProviderStartProvider,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Any | Error]:
    """Begin an OAuth login (redirect to the provider)

     Generates a CSRF `state` and a PKCE verifier, binds them into a short-lived signed httpOnly cookie
    (`recurso_oauth_state`, scoped to /auth/oauth), and 302-redirects to the provider's authorize URL.
    Unknown or disabled providers return 404.

    Args:
        provider (GetAuthOauthProviderStartProvider):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
    """

    kwargs = _get_kwargs(
        provider=provider,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    provider: GetAuthOauthProviderStartProvider,
    *,
    client: AuthenticatedClient | Client,
) -> Any | Error | None:
    """Begin an OAuth login (redirect to the provider)

     Generates a CSRF `state` and a PKCE verifier, binds them into a short-lived signed httpOnly cookie
    (`recurso_oauth_state`, scoped to /auth/oauth), and 302-redirects to the provider's authorize URL.
    Unknown or disabled providers return 404.

    Args:
        provider (GetAuthOauthProviderStartProvider):

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
        )
    ).parsed
