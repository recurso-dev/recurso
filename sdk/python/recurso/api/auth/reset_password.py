from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.reset_password_body import ResetPasswordBody
from ...models.reset_password_response_200 import ResetPasswordResponse200
from ...types import Response


def _get_kwargs(
    *,
    body: ResetPasswordBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/auth/reset-password",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | ResetPasswordResponse200 | None:
    if response.status_code == 200:
        response_200 = ResetPasswordResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | ResetPasswordResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: ResetPasswordBody,
) -> Response[Error | ResetPasswordResponse200]:
    """Reset a password with a token

     Consumes a valid (unused, unexpired) reset token, sets the new bcrypt password, and revokes ALL of
    that user's sessions (forces re-login everywhere). Invalid/expired/used tokens return a generic 400.

    Args:
        body (ResetPasswordBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ResetPasswordResponse200]
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
    body: ResetPasswordBody,
) -> Error | ResetPasswordResponse200 | None:
    """Reset a password with a token

     Consumes a valid (unused, unexpired) reset token, sets the new bcrypt password, and revokes ALL of
    that user's sessions (forces re-login everywhere). Invalid/expired/used tokens return a generic 400.

    Args:
        body (ResetPasswordBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ResetPasswordResponse200
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: ResetPasswordBody,
) -> Response[Error | ResetPasswordResponse200]:
    """Reset a password with a token

     Consumes a valid (unused, unexpired) reset token, sets the new bcrypt password, and revokes ALL of
    that user's sessions (forces re-login everywhere). Invalid/expired/used tokens return a generic 400.

    Args:
        body (ResetPasswordBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ResetPasswordResponse200]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: ResetPasswordBody,
) -> Error | ResetPasswordResponse200 | None:
    """Reset a password with a token

     Consumes a valid (unused, unexpired) reset token, sets the new bcrypt password, and revokes ALL of
    that user's sessions (forces re-login everywhere). Invalid/expired/used tokens return a generic 400.

    Args:
        body (ResetPasswordBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ResetPasswordResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
