from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.list_plans_response_200 import ListPlansResponse200
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    q: str | Unset = UNSET,
    limit: int | Unset = UNSET,
    page: int | Unset = 1,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["q"] = q

    params["limit"] = limit

    params["page"] = page

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/plans",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | ListPlansResponse200 | None:
    if response.status_code == 200:
        response_200 = ListPlansResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | ListPlansResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    limit: int | Unset = UNSET,
    page: int | Unset = 1,
) -> Response[Error | ListPlansResponse200]:
    """List plans

    Args:
        q (str | Unset):
        limit (int | Unset):
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ListPlansResponse200]
    """

    kwargs = _get_kwargs(
        q=q,
        limit=limit,
        page=page,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    limit: int | Unset = UNSET,
    page: int | Unset = 1,
) -> Error | ListPlansResponse200 | None:
    """List plans

    Args:
        q (str | Unset):
        limit (int | Unset):
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ListPlansResponse200
    """

    return sync_detailed(
        client=client,
        q=q,
        limit=limit,
        page=page,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    limit: int | Unset = UNSET,
    page: int | Unset = 1,
) -> Response[Error | ListPlansResponse200]:
    """List plans

    Args:
        q (str | Unset):
        limit (int | Unset):
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ListPlansResponse200]
    """

    kwargs = _get_kwargs(
        q=q,
        limit=limit,
        page=page,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    limit: int | Unset = UNSET,
    page: int | Unset = 1,
) -> Error | ListPlansResponse200 | None:
    """List plans

    Args:
        q (str | Unset):
        limit (int | Unset):
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ListPlansResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
            q=q,
            limit=limit,
            page=page,
        )
    ).parsed
