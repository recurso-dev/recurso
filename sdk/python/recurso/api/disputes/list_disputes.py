from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.list_disputes_response_200 import ListDisputesResponse200
from ...models.list_disputes_status import ListDisputesStatus
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    status: ListDisputesStatus | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/disputes",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | ListDisputesResponse200 | None:
    if response.status_code == 200:
        response_200 = ListDisputesResponse200.from_dict(response.json())

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
) -> Response[Error | ListDisputesResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    status: ListDisputesStatus | Unset = UNSET,
) -> Response[Error | ListDisputesResponse200]:
    """List invoice disputes (tenant-scoped)

     Lists disputes for the authenticated tenant. There is no admin dashboard UI for disputes yet; this
    API is the current surface.

    Args:
        status (ListDisputesStatus | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ListDisputesResponse200]
    """

    kwargs = _get_kwargs(
        status=status,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    status: ListDisputesStatus | Unset = UNSET,
) -> Error | ListDisputesResponse200 | None:
    """List invoice disputes (tenant-scoped)

     Lists disputes for the authenticated tenant. There is no admin dashboard UI for disputes yet; this
    API is the current surface.

    Args:
        status (ListDisputesStatus | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ListDisputesResponse200
    """

    return sync_detailed(
        client=client,
        status=status,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    status: ListDisputesStatus | Unset = UNSET,
) -> Response[Error | ListDisputesResponse200]:
    """List invoice disputes (tenant-scoped)

     Lists disputes for the authenticated tenant. There is no admin dashboard UI for disputes yet; this
    API is the current surface.

    Args:
        status (ListDisputesStatus | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ListDisputesResponse200]
    """

    kwargs = _get_kwargs(
        status=status,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    status: ListDisputesStatus | Unset = UNSET,
) -> Error | ListDisputesResponse200 | None:
    """List invoice disputes (tenant-scoped)

     Lists disputes for the authenticated tenant. There is no admin dashboard UI for disputes yet; this
    API is the current surface.

    Args:
        status (ListDisputesStatus | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ListDisputesResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
            status=status,
        )
    ).parsed
