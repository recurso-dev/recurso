from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.login_body import LoginBody
from ...models.login_response_200 import LoginResponse200
from ...types import Response


def _get_kwargs(
    *,
    body: LoginBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/auth/login",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | LoginResponse200 | None:
    if response.status_code == 200:
        response_200 = LoginResponse200.from_dict(response.json())

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
) -> Response[Error | LoginResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: LoginBody,
) -> Response[Error | LoginResponse200]:
    r"""Log in (dashboard user)

     Verifies email + password and opens a session (sets the httpOnly `recurso_session` cookie). Errors
    are deliberately generic (\"invalid credentials\") for both unknown email and wrong password — no
    user enumeration.

    Args:
        body (LoginBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | LoginResponse200]
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
    body: LoginBody,
) -> Error | LoginResponse200 | None:
    r"""Log in (dashboard user)

     Verifies email + password and opens a session (sets the httpOnly `recurso_session` cookie). Errors
    are deliberately generic (\"invalid credentials\") for both unknown email and wrong password — no
    user enumeration.

    Args:
        body (LoginBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | LoginResponse200
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: LoginBody,
) -> Response[Error | LoginResponse200]:
    r"""Log in (dashboard user)

     Verifies email + password and opens a session (sets the httpOnly `recurso_session` cookie). Errors
    are deliberately generic (\"invalid credentials\") for both unknown email and wrong password — no
    user enumeration.

    Args:
        body (LoginBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | LoginResponse200]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: LoginBody,
) -> Error | LoginResponse200 | None:
    r"""Log in (dashboard user)

     Verifies email + password and opens a session (sets the httpOnly `recurso_session` cookie). Errors
    are deliberately generic (\"invalid credentials\") for both unknown email and wrong password — no
    user enumeration.

    Args:
        body (LoginBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | LoginResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
