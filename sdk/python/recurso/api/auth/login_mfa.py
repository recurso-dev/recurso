from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.login_mfa_body import LoginMFABody
from ...models.login_mfa_response_200 import LoginMFAResponse200
from ...types import Response


def _get_kwargs(
    *,
    body: LoginMFABody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/auth/login/mfa",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | LoginMFAResponse200 | None:
    if response.status_code == 200:
        response_200 = LoginMFAResponse200.from_dict(response.json())

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
) -> Response[Error | LoginMFAResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: LoginMFABody,
) -> Response[Error | LoginMFAResponse200]:
    """Complete a two-step (MFA) login

     Second step of login for MFA-enabled users. Exchanges the short-lived single-use `mfa_token`
    returned by `/auth/login` (plus a TOTP code or an unused backup code) for a session (sets the
    httpOnly `recurso_session` cookie). The mfa_token and any backup code used are consumed. Errors are
    deliberately generic.

    Args:
        body (LoginMFABody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | LoginMFAResponse200]
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
    body: LoginMFABody,
) -> Error | LoginMFAResponse200 | None:
    """Complete a two-step (MFA) login

     Second step of login for MFA-enabled users. Exchanges the short-lived single-use `mfa_token`
    returned by `/auth/login` (plus a TOTP code or an unused backup code) for a session (sets the
    httpOnly `recurso_session` cookie). The mfa_token and any backup code used are consumed. Errors are
    deliberately generic.

    Args:
        body (LoginMFABody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | LoginMFAResponse200
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: LoginMFABody,
) -> Response[Error | LoginMFAResponse200]:
    """Complete a two-step (MFA) login

     Second step of login for MFA-enabled users. Exchanges the short-lived single-use `mfa_token`
    returned by `/auth/login` (plus a TOTP code or an unused backup code) for a session (sets the
    httpOnly `recurso_session` cookie). The mfa_token and any backup code used are consumed. Errors are
    deliberately generic.

    Args:
        body (LoginMFABody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | LoginMFAResponse200]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: LoginMFABody,
) -> Error | LoginMFAResponse200 | None:
    """Complete a two-step (MFA) login

     Second step of login for MFA-enabled users. Exchanges the short-lived single-use `mfa_token`
    returned by `/auth/login` (plus a TOTP code or an unused backup code) for a session (sets the
    httpOnly `recurso_session` cookie). The mfa_token and any backup code used are consumed. Errors are
    deliberately generic.

    Args:
        body (LoginMFABody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | LoginMFAResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
