from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.get_deferred_rollforward_response_200 import GetDeferredRollforwardResponse200
from ...types import UNSET, Response


def _get_kwargs(
    *,
    month: int,
    year: int,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["month"] = month

    params["year"] = year

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/ledger/deferred-rollforward",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | GetDeferredRollforwardResponse200 | None:
    if response.status_code == 200:
        response_200 = GetDeferredRollforwardResponse200.from_dict(response.json())

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
) -> Response[Error | GetDeferredRollforwardResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    month: int,
    year: int,
) -> Response[Error | GetDeferredRollforwardResponse200]:
    """Deferred-revenue rollforward

     The Deferred Revenue account's movement for a calendar month: opening
    balance, deferrals added, amounts released, and the derived closing
    (opening + added - released). Read-only.

    Args:
        month (int):
        year (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetDeferredRollforwardResponse200]
    """

    kwargs = _get_kwargs(
        month=month,
        year=year,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    month: int,
    year: int,
) -> Error | GetDeferredRollforwardResponse200 | None:
    """Deferred-revenue rollforward

     The Deferred Revenue account's movement for a calendar month: opening
    balance, deferrals added, amounts released, and the derived closing
    (opening + added - released). Read-only.

    Args:
        month (int):
        year (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetDeferredRollforwardResponse200
    """

    return sync_detailed(
        client=client,
        month=month,
        year=year,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    month: int,
    year: int,
) -> Response[Error | GetDeferredRollforwardResponse200]:
    """Deferred-revenue rollforward

     The Deferred Revenue account's movement for a calendar month: opening
    balance, deferrals added, amounts released, and the derived closing
    (opening + added - released). Read-only.

    Args:
        month (int):
        year (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetDeferredRollforwardResponse200]
    """

    kwargs = _get_kwargs(
        month=month,
        year=year,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    month: int,
    year: int,
) -> Error | GetDeferredRollforwardResponse200 | None:
    """Deferred-revenue rollforward

     The Deferred Revenue account's movement for a calendar month: opening
    balance, deferrals added, amounts released, and the derived closing
    (opening + added - released). Read-only.

    Args:
        month (int):
        year (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetDeferredRollforwardResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
            month=month,
            year=year,
        )
    ).parsed
