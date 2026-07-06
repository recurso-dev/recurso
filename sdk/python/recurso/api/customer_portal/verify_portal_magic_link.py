from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.verify_portal_magic_link_response_200 import VerifyPortalMagicLinkResponse200
from ...types import UNSET, Response


def _get_kwargs(
    *,
    token: str,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["token"] = token

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/portal/auth/verify",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | VerifyPortalMagicLinkResponse200 | None:
    if response.status_code == 200:
        response_200 = VerifyPortalMagicLinkResponse200.from_dict(response.json())

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
) -> Response[Error | VerifyPortalMagicLinkResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    token: str,
) -> Response[Error | VerifyPortalMagicLinkResponse200]:
    """Verify a magic link and start a session

     Exchanges the emailed token for a 7-day portal session. The session token is also set as the
    `portal_session` cookie.

    Args:
        token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VerifyPortalMagicLinkResponse200]
    """

    kwargs = _get_kwargs(
        token=token,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    token: str,
) -> Error | VerifyPortalMagicLinkResponse200 | None:
    """Verify a magic link and start a session

     Exchanges the emailed token for a 7-day portal session. The session token is also set as the
    `portal_session` cookie.

    Args:
        token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VerifyPortalMagicLinkResponse200
    """

    return sync_detailed(
        client=client,
        token=token,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    token: str,
) -> Response[Error | VerifyPortalMagicLinkResponse200]:
    """Verify a magic link and start a session

     Exchanges the emailed token for a 7-day portal session. The session token is also set as the
    `portal_session` cookie.

    Args:
        token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VerifyPortalMagicLinkResponse200]
    """

    kwargs = _get_kwargs(
        token=token,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    token: str,
) -> Error | VerifyPortalMagicLinkResponse200 | None:
    """Verify a magic link and start a session

     Exchanges the emailed token for a 7-day portal session. The session token is also set as the
    `portal_session` cookie.

    Args:
        token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VerifyPortalMagicLinkResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
            token=token,
        )
    ).parsed
