from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.api_key import APIKey
from ...models.create_api_key_body import CreateAPIKeyBody
from ...models.error import Error
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    body: CreateAPIKeyBody | Unset = UNSET,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/developer/keys",
    }

    if not isinstance(body, Unset):
        _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> APIKey | Error | None:
    if response.status_code == 201:
        response_201 = APIKey.from_dict(response.json())

        return response_201

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[APIKey | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: CreateAPIKeyBody | Unset = UNSET,
) -> Response[APIKey | Error]:
    """Create an API key

     Generates a new secret key. The raw `key_value` is returned only in this response. Choose the mode
    with `mode`: a `test` key (rsk_test_, the default) works on a non-live server; a `live` key
    (rsk_live_) works only on a server configured with live payment gateways.

    Args:
        body (CreateAPIKeyBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[APIKey | Error]
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
    body: CreateAPIKeyBody | Unset = UNSET,
) -> APIKey | Error | None:
    """Create an API key

     Generates a new secret key. The raw `key_value` is returned only in this response. Choose the mode
    with `mode`: a `test` key (rsk_test_, the default) works on a non-live server; a `live` key
    (rsk_live_) works only on a server configured with live payment gateways.

    Args:
        body (CreateAPIKeyBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        APIKey | Error
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: CreateAPIKeyBody | Unset = UNSET,
) -> Response[APIKey | Error]:
    """Create an API key

     Generates a new secret key. The raw `key_value` is returned only in this response. Choose the mode
    with `mode`: a `test` key (rsk_test_, the default) works on a non-live server; a `live` key
    (rsk_live_) works only on a server configured with live payment gateways.

    Args:
        body (CreateAPIKeyBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[APIKey | Error]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: CreateAPIKeyBody | Unset = UNSET,
) -> APIKey | Error | None:
    """Create an API key

     Generates a new secret key. The raw `key_value` is returned only in this response. Choose the mode
    with `mode`: a `test` key (rsk_test_, the default) works on a non-live server; a `live` key
    (rsk_live_) works only on a server configured with live payment gateways.

    Args:
        body (CreateAPIKeyBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        APIKey | Error
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
